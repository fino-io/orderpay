package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/pagination"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	domainrepo "github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/repo"
	mapper "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo/mapper"
	record "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo/model/sqlite"
	"gorm.io/gorm"
)

type WorkflowStore struct {
	db    *gorm.DB
	codec *pagination.CursorCodec
}

type workflowTx struct {
	db *gorm.DB
}

func NewWorkflowStore(db *gorm.DB, codec *pagination.CursorCodec) *WorkflowStore {
	return &WorkflowStore{
		db:    db,
		codec: codec,
	}
}

func (s *WorkflowStore) Transaction(ctx context.Context, fn func(domainrepo.WorkflowTx) error) error {
	return s.transaction(ctx, "BEGIN IMMEDIATE", func(tx *workflowTx) error {
		return fn(tx)
	})
}

func (s *WorkflowStore) View(ctx context.Context, fn func(domainrepo.WorkflowReader) error) error {
	return s.transaction(ctx, "BEGIN", func(tx *workflowTx) error {
		return fn(tx)
	})
}

func (s *WorkflowStore) transaction(ctx context.Context, begin string, fn func(*workflowTx) error) error {
	return s.db.WithContext(ctx).Connection(func(conn *gorm.DB) (txErr error) {
		if err := conn.Exec(begin).Error; err != nil {
			return err
		}

		committed := false
		defer func() {
			if !committed {
				cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
				defer cancel()
				txErr = errors.Join(txErr, conn.WithContext(cleanupCtx).Exec("ROLLBACK").Error)
			}
		}()

		if err := fn(&workflowTx{db: conn.Session(&gorm.Session{SkipDefaultTransaction: true})}); err != nil {
			return err
		}

		if err := conn.Exec("COMMIT").Error; err != nil {
			return err
		}
		committed = true
		return nil
	})
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return core.NewNotFoundError("resource not found")
	}
	return err
}

func (t *workflowTx) Order(merchant, id string) (*model.Order, error) {
	var row record.Order
	if err := t.db.Where("merchant_id = ? AND id = ?", merchant, id).First(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return mapper.RecordToOrder(&row), nil
}

func (t *workflowTx) OrderByNumber(merchant, number string) (*model.Order, error) {
	var row record.Order
	if err := t.db.Where("merchant_id = ? AND merchant_order_no = ?", merchant, number).First(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return mapper.RecordToOrder(&row), nil
}

func (t *workflowTx) Payment(merchant, id string) (*model.Payment, error) {
	var row record.Payment
	if err := t.db.Where("merchant_id = ? AND order_id = ?", merchant, id).First(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return mapper.RecordToPayment(&row), nil
}

func (t *workflowTx) SaveOrder(entity *model.Order) error {
	return t.db.Save(mapper.OrderToRecord(entity)).Error
}

func (t *workflowTx) SavePayment(entity *model.Payment) error {
	return t.db.Save(mapper.PaymentToRecord(entity)).Error
}

func (t *workflowTx) AppendAudit(entity *model.AuditEvent) error {
	return t.db.Create(mapper.AuditEventToRecord(entity)).Error
}

func (s *WorkflowStore) Orders(ctx context.Context, merchant string, query model.ListOrderQuery) (model.ListOrderResult, error) {
	scope := struct {
		Merchant string
		Status   int32
	}{
		Merchant: merchant,
		Status:   query.Filter.Status,
	}
	after, size, binding, err := pagination.ResolveKeyset(s.codec, query.PageToken, query.PageSize, "merchant_orders", scope)
	if err != nil {
		return model.ListOrderResult{}, core.NewBadRequestError("invalid page token")
	}

	q := s.db.WithContext(ctx).Where("merchant_id = ?", merchant)
	if query.Filter.Status != 0 {
		q = q.Where("status = ?", query.Filter.Status)
	}
	if after != "" {
		var anchor record.Order
		if err := s.db.WithContext(ctx).Where("merchant_id = ? AND id = ?", merchant, after).First(&anchor).Error; err != nil {
			return model.ListOrderResult{}, core.NewBadRequestError("invalid cursor")
		}
		q = q.Where("create_time > ? OR (create_time = ? AND id > ?)", anchor.CreateTime, anchor.CreateTime, after)
	}

	var rows []*record.Order
	if err := q.Order("create_time ASC, id ASC").Limit(size + 1).Find(&rows).Error; err != nil {
		return model.ListOrderResult{}, err
	}

	next := ""
	if len(rows) > size {
		rows = rows[:size]
		next, err = s.codec.EncodeKeyset(string(rows[len(rows)-1].ID), binding)
		if err != nil {
			return model.ListOrderResult{}, err
		}
	}
	return model.ListOrderResult{
		Items:         mapper.RecordsToOrders(rows),
		NextPageToken: next,
	}, nil
}

func (s *WorkflowStore) Audits(ctx context.Context, merchant, resource string, requested int, token string) (model.ListAuditEventResult, error) {
	scope := struct {
		Merchant string
		Resource string
	}{
		Merchant: merchant,
		Resource: resource,
	}
	after, size, binding, err := pagination.ResolveKeyset(s.codec, token, requested, "admin_audits", scope)
	if err != nil {
		return model.ListAuditEventResult{}, core.NewBadRequestError("invalid page token")
	}

	q := s.db.WithContext(ctx).Model(&record.AuditEvent{})
	if merchant != "" {
		q = q.Where("merchant_id = ?", merchant)
	}
	if resource != "" {
		q = q.Where("resource_id = ?", resource)
	}
	if after != "" {
		var anchor record.AuditEvent
		anchorQuery := s.db.WithContext(ctx).Where("id = ?", after)
		if merchant != "" {
			anchorQuery = anchorQuery.Where("merchant_id = ?", merchant)
		}
		if resource != "" {
			anchorQuery = anchorQuery.Where("resource_id = ?", resource)
		}
		if err := anchorQuery.First(&anchor).Error; err != nil {
			return model.ListAuditEventResult{}, core.NewBadRequestError("invalid cursor")
		}
		q = q.Where("create_time > ? OR (create_time = ? AND id > ?)", anchor.CreateTime, anchor.CreateTime, after)
	}

	var rows []*record.AuditEvent
	if err := q.Order("create_time ASC, id ASC").Limit(size + 1).Find(&rows).Error; err != nil {
		return model.ListAuditEventResult{}, err
	}

	next := ""
	if len(rows) > size {
		rows = rows[:size]
		next, err = s.codec.EncodeKeyset(string(rows[len(rows)-1].ID), binding)
		if err != nil {
			return model.ListAuditEventResult{}, err
		}
	}
	return model.ListAuditEventResult{
		Items:         mapper.RecordsToAuditEvents(rows),
		NextPageToken: next,
	}, nil
}

func (s *WorkflowStore) Initialize() error {
	for _, statement := range []string{
		`CREATE TRIGGER IF NOT EXISTS audit_no_update BEFORE UPDATE ON audit_event BEGIN SELECT RAISE(ABORT, 'audit append only'); END`,
		`CREATE TRIGGER IF NOT EXISTS audit_no_delete BEFORE DELETE ON audit_event BEGIN SELECT RAISE(ABORT, 'audit append only'); END`,
		`CREATE TRIGGER IF NOT EXISTS order_amount_insert BEFORE INSERT ON "order" WHEN NEW.amount_minor < 1 OR NEW.amount_minor > 100000000 BEGIN SELECT RAISE(ABORT, 'invalid amount'); END`,
		`CREATE TRIGGER IF NOT EXISTS order_content_immutable BEFORE UPDATE ON "order" WHEN NEW.amount_minor != OLD.amount_minor OR NEW.merchant_id != OLD.merchant_id OR NEW.merchant_order_no != OLD.merchant_order_no OR NEW.description != OLD.description BEGIN SELECT RAISE(ABORT, 'order content immutable'); END`,
		`CREATE TRIGGER IF NOT EXISTS payment_order_insert BEFORE INSERT ON payment WHEN NOT EXISTS (SELECT 1 FROM "order" WHERE id=NEW.order_id AND merchant_id=NEW.merchant_id AND amount_minor=NEW.amount_minor) BEGIN SELECT RAISE(ABORT, 'payment order mismatch'); END`,
		`CREATE TRIGGER IF NOT EXISTS payment_status_insert BEFORE INSERT ON payment WHEN NEW.status NOT IN (1,2,3,4) BEGIN SELECT RAISE(ABORT, 'invalid payment status'); END`,
		`CREATE TRIGGER IF NOT EXISTS order_status_insert BEFORE INSERT ON "order" WHEN NEW.status NOT IN (1,2,3) BEGIN SELECT RAISE(ABORT, 'invalid order status'); END`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_provider ON payment(channel_account_id,provider_reference) WHERE provider_reference != ''`,
	} {
		if err := s.db.Exec(statement).Error; err != nil {
			return fmt.Errorf("initialize workflow constraints: %w", err)
		}
	}
	return nil
}
