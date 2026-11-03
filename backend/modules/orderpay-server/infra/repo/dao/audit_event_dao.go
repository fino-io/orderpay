package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/db"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	pmodel "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo/model/sqlite"
	"gorm.io/gorm"
)

//go:generate mockgen -destination=mocks/audit_event_dao.go -package=mocks . IAuditEventDao
type IAuditEventDao interface {
	Get(ctx context.Context, id model.AuditEventID, opts ...db.Option) (*pmodel.AuditEvent, error)
	List(ctx context.Context, filter model.AuditEventFilter, afterID model.AuditEventID, limit int, opts ...db.Option) ([]*pmodel.AuditEvent, error)
	Count(ctx context.Context, filter model.AuditEventFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.AuditEvent, opts ...db.Option) (model.AuditEventID, error)
	Update(ctx context.Context, id model.AuditEventID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.AuditEventID, opts ...db.Option) error
}

type AuditEventDaoImpl struct {
	p db.Provider
}

func NewAuditEventDaoImpl(p db.Provider) IAuditEventDao {
	return &AuditEventDaoImpl{
		p: p,
	}
}

func AutoMigrateAuditEvent(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.AuditEvent{}); err != nil {
		return fmt.Errorf("auto migrate audit_event model: %w", err)
	}

	return nil
}

func (m *AuditEventDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.AuditEvent{})
	return tx
}

func (m *AuditEventDaoImpl) Get(ctx context.Context, id model.AuditEventID, opts ...db.Option) (*pmodel.AuditEvent, error) {
	record := &pmodel.AuditEvent{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("audit_event %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *AuditEventDaoImpl) Create(ctx context.Context, record *pmodel.AuditEvent, opts ...db.Option) (model.AuditEventID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("audit_event already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *AuditEventDaoImpl) filteredQuery(ctx context.Context, filter model.AuditEventFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	return tx
}

func (m *AuditEventDaoImpl) List(ctx context.Context, filter model.AuditEventFilter, afterID model.AuditEventID, limit int, opts ...db.Option) ([]*pmodel.AuditEvent, error) {
	var records []*pmodel.AuditEvent
	tx := m.filteredQuery(ctx, filter, opts...)
	tx = tx.Order("id DESC")
	if afterID != "" {
		tx = tx.Where("id < ?", afterID)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (m *AuditEventDaoImpl) Count(ctx context.Context, filter model.AuditEventFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *AuditEventDaoImpl) Update(ctx context.Context, id model.AuditEventID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("audit_event already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("audit_event %s not found", id)
	}
	return nil
}

func (m *AuditEventDaoImpl) Delete(ctx context.Context, id model.AuditEventID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.AuditEvent{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("audit_event %s not found", id)
	}
	return nil
}
