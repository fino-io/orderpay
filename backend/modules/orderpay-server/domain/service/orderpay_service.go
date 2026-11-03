package service

import (
	"context"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/repo"
)

//go:generate mockgen -destination=mocks/orderpay_service.go -package=mocks . IOrderpayService
type IOrderpayService interface {
	GetAuditEvent(ctx context.Context, id model.AuditEventID) (*model.AuditEvent, error)
	ListAuditEvents(ctx context.Context, query model.ListAuditEventQuery) (model.ListAuditEventResult, error)
	CreateAuditEvent(ctx context.Context, input model.CreateAuditEventInput) (*model.AuditEvent, error)
	UpdateAuditEvent(ctx context.Context, id model.AuditEventID, input model.UpdateAuditEventInput) (*model.AuditEvent, error)
	DeleteAuditEvent(ctx context.Context, id model.AuditEventID) error

	GetOrder(ctx context.Context, id model.OrderID) (*model.Order, error)
	ListOrders(ctx context.Context, query model.ListOrderQuery) (model.ListOrderResult, error)
	CreateOrder(ctx context.Context, input model.CreateOrderInput) (*model.Order, error)
	UpdateOrder(ctx context.Context, id model.OrderID, input model.UpdateOrderInput) (*model.Order, error)
	DeleteOrder(ctx context.Context, id model.OrderID) error

	GetPayment(ctx context.Context, id model.PaymentID) (*model.Payment, error)
	ListPayments(ctx context.Context, query model.ListPaymentQuery) (model.ListPaymentResult, error)
	CreatePayment(ctx context.Context, input model.CreatePaymentInput) (*model.Payment, error)
	UpdatePayment(ctx context.Context, id model.PaymentID, input model.UpdatePaymentInput) (*model.Payment, error)
	DeletePayment(ctx context.Context, id model.PaymentID) error
}

type ServiceDeps struct {
	AuditEventRepo repo.IAuditEventRepo
	OrderRepo      repo.IOrderRepo
	PaymentRepo    repo.IPaymentRepo
}

type OrderpayServiceImpl struct {
	auditEventRepo repo.IAuditEventRepo
	orderRepo      repo.IOrderRepo
	paymentRepo    repo.IPaymentRepo
}

func NewOrderpayService(deps ServiceDeps) IOrderpayService {
	return &OrderpayServiceImpl{
		auditEventRepo: deps.AuditEventRepo,
		orderRepo:      deps.OrderRepo,
		paymentRepo:    deps.PaymentRepo,
	}
}

func (s *OrderpayServiceImpl) GetAuditEvent(ctx context.Context, id model.AuditEventID) (*model.AuditEvent, error) {
	return s.auditEventRepo.GetAuditEvent(ctx, id)
}

func (s *OrderpayServiceImpl) ListAuditEvents(ctx context.Context, query model.ListAuditEventQuery) (model.ListAuditEventResult, error) {
	return s.auditEventRepo.ListAuditEvents(ctx, query)
}

func (s *OrderpayServiceImpl) CreateAuditEvent(ctx context.Context, input model.CreateAuditEventInput) (*model.AuditEvent, error) {
	// TODO: set MerchantId, ActorId, Action, ResourceId, RequestId, BeforeStatus, AfterStatus, ReasonCode, EvidenceReference before creating AuditEvent.
	entity, err := model.NewAuditEvent(input)
	if err != nil {
		return nil, err
	}

	return s.auditEventRepo.CreateAuditEvent(ctx, entity)
}

func (s *OrderpayServiceImpl) UpdateAuditEvent(ctx context.Context, id model.AuditEventID, input model.UpdateAuditEventInput) (*model.AuditEvent, error) {
	entity, err := s.auditEventRepo.GetAuditEvent(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := entity.ApplyUpdate(input); err != nil {
		return nil, err
	}

	if err := s.auditEventRepo.UpdateAuditEvent(ctx, entity, input.UpdateMask); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *OrderpayServiceImpl) DeleteAuditEvent(ctx context.Context, id model.AuditEventID) error {
	return s.auditEventRepo.DeleteAuditEvent(ctx, id)
}

func (s *OrderpayServiceImpl) GetOrder(ctx context.Context, id model.OrderID) (*model.Order, error) {
	return s.orderRepo.GetOrder(ctx, id)
}

func (s *OrderpayServiceImpl) ListOrders(ctx context.Context, query model.ListOrderQuery) (model.ListOrderResult, error) {
	return s.orderRepo.ListOrders(ctx, query)
}

func (s *OrderpayServiceImpl) CreateOrder(ctx context.Context, input model.CreateOrderInput) (*model.Order, error) {
	// TODO: set MerchantId, Status before creating Order.
	entity, err := model.NewOrder(input)
	if err != nil {
		return nil, err
	}

	return s.orderRepo.CreateOrder(ctx, entity)
}

func (s *OrderpayServiceImpl) UpdateOrder(ctx context.Context, id model.OrderID, input model.UpdateOrderInput) (*model.Order, error) {
	entity, err := s.orderRepo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(input.UpdateMask) > 0 {
		input.UpdateTime = core.Now()
		input.UpdateMask = append(input.UpdateMask, "updateTime")
	}

	if err := entity.ApplyUpdate(input); err != nil {
		return nil, err
	}

	if err := s.orderRepo.UpdateOrder(ctx, entity, input.UpdateMask); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *OrderpayServiceImpl) DeleteOrder(ctx context.Context, id model.OrderID) error {
	return s.orderRepo.DeleteOrder(ctx, id)
}

func (s *OrderpayServiceImpl) GetPayment(ctx context.Context, id model.PaymentID) (*model.Payment, error) {
	return s.paymentRepo.GetPayment(ctx, id)
}

func (s *OrderpayServiceImpl) ListPayments(ctx context.Context, query model.ListPaymentQuery) (model.ListPaymentResult, error) {
	return s.paymentRepo.ListPayments(ctx, query)
}

func (s *OrderpayServiceImpl) CreatePayment(ctx context.Context, input model.CreatePaymentInput) (*model.Payment, error) {
	// TODO: set MerchantId, OrderId, AmountMinor, ChannelAccountId, Status, ProviderReference, ReasonCode, SubmittedAt, RetryDeadlineAt, SubmitAttempts, NextCheckAt, CheckAttempts before creating Payment.
	entity, err := model.NewPayment(input)
	if err != nil {
		return nil, err
	}

	return s.paymentRepo.CreatePayment(ctx, entity)
}

func (s *OrderpayServiceImpl) UpdatePayment(ctx context.Context, id model.PaymentID, input model.UpdatePaymentInput) (*model.Payment, error) {
	entity, err := s.paymentRepo.GetPayment(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(input.UpdateMask) > 0 {
		input.UpdateTime = core.Now()
		input.UpdateMask = append(input.UpdateMask, "updateTime")
	}

	if err := entity.ApplyUpdate(input); err != nil {
		return nil, err
	}

	if err := s.paymentRepo.UpdatePayment(ctx, entity, input.UpdateMask); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *OrderpayServiceImpl) DeletePayment(ctx context.Context, id model.PaymentID) error {
	return s.paymentRepo.DeletePayment(ctx, id)
}
