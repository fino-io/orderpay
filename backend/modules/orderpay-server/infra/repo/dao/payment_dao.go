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

//go:generate mockgen -destination=mocks/payment_dao.go -package=mocks . IPaymentDao
type IPaymentDao interface {
	Get(ctx context.Context, id model.PaymentID, opts ...db.Option) (*pmodel.Payment, error)
	List(ctx context.Context, filter model.PaymentFilter, afterID model.PaymentID, limit int, opts ...db.Option) ([]*pmodel.Payment, error)
	Count(ctx context.Context, filter model.PaymentFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.Payment, opts ...db.Option) (model.PaymentID, error)
	Update(ctx context.Context, id model.PaymentID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.PaymentID, opts ...db.Option) error
}

type PaymentDaoImpl struct {
	p db.Provider
}

func NewPaymentDaoImpl(p db.Provider) IPaymentDao {
	return &PaymentDaoImpl{
		p: p,
	}
}

func AutoMigratePayment(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.Payment{}); err != nil {
		return fmt.Errorf("auto migrate payment model: %w", err)
	}

	return nil
}

func (m *PaymentDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.Payment{})
	return tx
}

func (m *PaymentDaoImpl) Get(ctx context.Context, id model.PaymentID, opts ...db.Option) (*pmodel.Payment, error) {
	record := &pmodel.Payment{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("payment %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *PaymentDaoImpl) Create(ctx context.Context, record *pmodel.Payment, opts ...db.Option) (model.PaymentID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("payment already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *PaymentDaoImpl) filteredQuery(ctx context.Context, filter model.PaymentFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	return tx
}

func (m *PaymentDaoImpl) List(ctx context.Context, filter model.PaymentFilter, afterID model.PaymentID, limit int, opts ...db.Option) ([]*pmodel.Payment, error) {
	var records []*pmodel.Payment
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

func (m *PaymentDaoImpl) Count(ctx context.Context, filter model.PaymentFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *PaymentDaoImpl) Update(ctx context.Context, id model.PaymentID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("payment already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("payment %s not found", id)
	}
	return nil
}

func (m *PaymentDaoImpl) Delete(ctx context.Context, id model.PaymentID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.Payment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("payment %s not found", id)
	}
	return nil
}
