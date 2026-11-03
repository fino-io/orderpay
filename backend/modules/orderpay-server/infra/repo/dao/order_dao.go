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

//go:generate mockgen -destination=mocks/order_dao.go -package=mocks . IOrderDao
type IOrderDao interface {
	Get(ctx context.Context, id model.OrderID, opts ...db.Option) (*pmodel.Order, error)
	List(ctx context.Context, filter model.OrderFilter, afterID model.OrderID, limit int, opts ...db.Option) ([]*pmodel.Order, error)
	Count(ctx context.Context, filter model.OrderFilter, opts ...db.Option) (int64, error)
	Create(ctx context.Context, record *pmodel.Order, opts ...db.Option) (model.OrderID, error)
	Update(ctx context.Context, id model.OrderID, fields map[string]interface{}, opts ...db.Option) error
	Delete(ctx context.Context, id model.OrderID, opts ...db.Option) error
}

type OrderDaoImpl struct {
	p db.Provider
}

func NewOrderDaoImpl(p db.Provider) IOrderDao {
	return &OrderDaoImpl{
		p: p,
	}
}

func AutoMigrateOrder(provider db.Provider) error {
	if provider == nil {
		return fmt.Errorf("db provider is nil")
	}

	gdb := provider.NewSession(context.Background())
	if gdb == nil {
		return fmt.Errorf("gorm DB is nil")
	}

	if err := gdb.AutoMigrate(&pmodel.Order{}); err != nil {
		return fmt.Errorf("auto migrate order model: %w", err)
	}

	return nil
}

func (m *OrderDaoImpl) baseQuery(ctx context.Context, opts ...db.Option) *gorm.DB {
	tx := m.p.NewSession(ctx, opts...).Model(&pmodel.Order{})
	return tx
}

func (m *OrderDaoImpl) Get(ctx context.Context, id model.OrderID, opts ...db.Option) (*pmodel.Order, error) {
	record := &pmodel.Order{}
	err := m.baseQuery(ctx, opts...).First(record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.NewNotFoundError("order %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (m *OrderDaoImpl) Create(ctx context.Context, record *pmodel.Order, opts ...db.Option) (model.OrderID, error) {
	result := m.p.NewSession(ctx, opts...).Create(record)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return "", core.NewAlreadyExistsError("order already exists")
	}
	if result.Error != nil {
		return "", result.Error
	}
	return record.ID, nil
}

func (m *OrderDaoImpl) filteredQuery(ctx context.Context, filter model.OrderFilter, opts ...db.Option) *gorm.DB {
	tx := m.baseQuery(ctx, opts...)
	if len(filter.IDs) > 0 {
		tx = tx.Where("id in ?", filter.IDs)
	}
	if filter.Status != 0 {
		tx = tx.Where("status = ?", filter.Status)
	}
	return tx
}

func (m *OrderDaoImpl) List(ctx context.Context, filter model.OrderFilter, afterID model.OrderID, limit int, opts ...db.Option) ([]*pmodel.Order, error) {
	var records []*pmodel.Order
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

func (m *OrderDaoImpl) Count(ctx context.Context, filter model.OrderFilter, opts ...db.Option) (int64, error) {
	tx := m.filteredQuery(ctx, filter, opts...)
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (m *OrderDaoImpl) Update(ctx context.Context, id model.OrderID, fields map[string]interface{}, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Updates(fields)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return core.NewAlreadyExistsError("order already exists")
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("order %s not found", id)
	}
	return nil
}

func (m *OrderDaoImpl) Delete(ctx context.Context, id model.OrderID, opts ...db.Option) error {
	result := m.baseQuery(ctx, opts...).Where("id = ?", id).Delete(&pmodel.Order{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return core.NewNotFoundError("order %s not found", id)
	}
	return nil
}
