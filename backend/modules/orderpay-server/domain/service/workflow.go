package service

import (
	"context"
	"unicode/utf8"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/repo"
	pb "github.com/fino-io/orderpay/go/orderpay"
)

type MandateVerifier interface {
	Authorize(context.Context, string, string) (string, error)
}

type WriteLimiter interface {
	Allow(context.Context, string) (bool, error)
}

type Workflow struct {
	writes   WriteLimiter
	store    repo.WorkflowStore
	mandates MandateVerifier
}

func NewWorkflow(store repo.WorkflowStore, mandates MandateVerifier, writes WriteLimiter) *Workflow {
	return &Workflow{
		store:    store,
		writes:   writes,
		mandates: mandates,
	}
}

func (s *Workflow) allowWrite(ctx context.Context, merchant string) error {
	if s.writes == nil {
		return model.ErrUnavailable
	}
	allowed, err := s.writes.Allow(ctx, merchant)
	if err != nil {
		return err
	}
	if !allowed {
		return model.ErrRateLimited
	}
	return nil
}

func (s *Workflow) audit(ctx context.Context, tx repo.WorkflowTx, action, resource, before, after, reason string) error {
	principal := model.PrincipalFrom(ctx)
	event, err := model.NewAuditEvent(model.CreateAuditEventInput{
		MerchantId:   principal.MerchantID,
		ActorId:      principal.ActorID,
		RequestId:    principal.RequestID,
		Action:       action,
		ResourceId:   resource,
		BeforeStatus: before,
		AfterStatus:  after,
		ReasonCode:   reason,
	})
	if err != nil {
		return err
	}
	return tx.AppendAudit(event)
}

func (s *Workflow) CreateOrder(ctx context.Context, input model.CreateOrderInput) (*model.Order, error) {
	if input.MerchantOrderNo == "" || utf8.RuneCountInString(input.MerchantOrderNo) > 128 || input.AmountMinor < 1 || input.AmountMinor > 100000000 || input.Description == "" || utf8.RuneCountInString(input.Description) > 200 {
		return nil, core.NewBadRequestError("invalid order input")
	}

	principal := model.PrincipalFrom(ctx)
	input.MerchantId = principal.MerchantID
	input.Status = pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT
	var result *model.Order
	err := s.store.Transaction(ctx, func(tx repo.WorkflowTx) error {
		existing, err := tx.OrderByNumber(principal.MerchantID, input.MerchantOrderNo)
		if err != nil && !core.IsNotFoundError(err) {
			return err
		}
		if existing != nil {
			if existing.AmountMinor() != input.AmountMinor || existing.Description() != input.Description {
				return model.ErrConflict
			}
			result = existing
			return nil
		}
		if err := s.allowWrite(ctx, principal.MerchantID); err != nil {
			return err
		}

		result, err = model.NewOrder(input)
		if err != nil {
			return err
		}
		if err := tx.SaveOrder(result); err != nil {
			return err
		}
		return s.audit(ctx, tx, "ORDER_CREATED", string(result.ID()), "", input.Status.String(), "")
	})
	return result, err
}

func (s *Workflow) GetOrder(ctx context.Context, id string) (*model.Order, *model.Payment, error) {
	var order *model.Order
	var payment *model.Payment
	err := s.store.Transaction(ctx, func(tx repo.WorkflowTx) error {
		var err error
		merchant := model.PrincipalFrom(ctx).MerchantID
		order, err = tx.Order(merchant, id)
		if err != nil {
			return err
		}
		payment, err = tx.Payment(merchant, id)
		if core.IsNotFoundError(err) {
			return nil
		}
		return err
	})
	return order, payment, err
}

func (s *Workflow) ListOrders(ctx context.Context, query model.ListOrderQuery) (model.ListOrderResult, error) {
	return s.store.Orders(ctx, model.PrincipalFrom(ctx).MerchantID, query)
}

func (s *Workflow) CancelOrder(ctx context.Context, id string) (*model.Order, error) {
	var result *model.Order
	err := s.store.Transaction(ctx, func(tx repo.WorkflowTx) error {
		merchant := model.PrincipalFrom(ctx).MerchantID
		order, err := tx.Order(merchant, id)
		if err != nil {
			return err
		}
		result = order
		if order.Status() == pb.OrderStatus_ORDER_STATUS_CANCELLED {
			return nil
		}
		if order.Status() != pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT {
			return model.ErrConflict
		}
		payment, err := tx.Payment(merchant, id)
		if err != nil && !core.IsNotFoundError(err) {
			return err
		}
		if err := s.allowWrite(ctx, merchant); err != nil {
			return err
		}

		if payment != nil {
			if payment.ReasonCode() == "CHANNEL_RESULT_CONFLICT" || payment.ReasonCode() == "COMPENSATION_REQUIRED" || payment.ReasonCode() == "RECONCILIATION_REQUIRED" {
				return model.ErrConflict
			}
			switch payment.Status() {
			case pb.PaymentStatus_PAYMENT_STATUS_CREATED:
				snapshot := payment.Snapshot()
				if snapshot.SubmitAttempts != 0 || snapshot.SubmittedAt != nil {
					return model.ErrConflict
				}
				before := snapshot.Status.String()
				snapshot.Status = pb.PaymentStatus_PAYMENT_STATUS_DECLINED
				snapshot.ReasonCode = "ORDER_CANCELLED"
				snapshot.NextCheckAt = nil
				snapshot.UpdateTime = core.Now()
				if err := tx.SavePayment(model.RebuildPayment(snapshot)); err != nil {
					return err
				}
				if err := s.audit(ctx, tx, "PAYMENT_CANCELLED", string(payment.ID()), before, snapshot.Status.String(), "ORDER_CANCELLED"); err != nil {
					return err
				}
			case pb.PaymentStatus_PAYMENT_STATUS_DECLINED:
			default:
				return model.ErrConflict
			}
		}
		snapshot := order.Snapshot()
		before := snapshot.Status.String()
		snapshot.Status = pb.OrderStatus_ORDER_STATUS_CANCELLED
		snapshot.UpdateTime = core.Now()
		result = model.RebuildOrder(snapshot)
		if err := tx.SaveOrder(result); err != nil {
			return err
		}
		return s.audit(ctx, tx, "ORDER_CANCELLED", id, before, snapshot.Status.String(), "")
	})
	return result, err
}

func (s *Workflow) CreatePayment(ctx context.Context, id, mandate string) (*model.Payment, error) {
	if mandate == "" || utf8.RuneCountInString(mandate) > 128 {
		return nil, core.NewBadRequestError("invalid mandate reference")
	}

	order, existing, err := s.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.MandateReference() != mandate {
			return nil, model.ErrConflict
		}
		return existing, nil
	}
	if order.Status() != pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT {
		return nil, model.ErrConflict
	}
	if s.mandates == nil {
		return nil, model.ErrUnavailable
	}

	merchant := model.PrincipalFrom(ctx).MerchantID
	account, err := s.mandates.Authorize(ctx, merchant, mandate)
	if err != nil {
		return nil, err
	}
	if account == "" {
		return nil, model.ErrUnavailable
	}

	var result *model.Payment
	err = s.store.Transaction(ctx, func(tx repo.WorkflowTx) error {
		current, err := tx.Order(merchant, id)
		if err != nil {
			return err
		}
		result, err = tx.Payment(merchant, id)
		if err == nil {
			if result.MandateReference() != mandate {
				return model.ErrConflict
			}
			return nil
		}
		if !core.IsNotFoundError(err) {
			return err
		}
		if current.Status() != pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT {
			return model.ErrConflict
		}

		if err := s.allowWrite(ctx, merchant); err != nil {
			return err
		}

		result, err = model.NewPayment(model.CreatePaymentInput{
			MerchantId:       merchant,
			OrderId:          id,
			AmountMinor:      current.AmountMinor(),
			MandateReference: mandate,
			ChannelAccountId: account,
			Status:           pb.PaymentStatus_PAYMENT_STATUS_CREATED,
			NextCheckAt:      core.Now(),
		})
		if err != nil {
			return err
		}
		if err := tx.SavePayment(result); err != nil {
			return err
		}
		return s.audit(ctx, tx, "PAYMENT_ACCEPTED", string(result.ID()), "", result.Status().String(), "")
	})
	return result, err
}

func (s *Workflow) ListAudits(ctx context.Context, merchant, resource string, size int, token string) (model.ListAuditEventResult, error) {
	result, err := s.store.Audits(ctx, merchant, resource, size, token)
	if err != nil {
		return result, err
	}
	err = s.store.Transaction(ctx, func(tx repo.WorkflowTx) error {
		return s.audit(ctx, tx, "AUDIT_QUERIED", resource, "", "", "")
	})
	return result, err
}
