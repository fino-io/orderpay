package repo

import (
	"context"
	"fmt"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/pagination"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	domainrepo "github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/repo"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo/dao"
	repomapper "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo/mapper"
)

var _ domainrepo.IOrderRepo = (*OrderRepoImpl)(nil)

type OrderRepoDeps struct {
	OrderDao       dao.IOrderDao
	PageTokenCodec *pagination.CursorCodec
}

type OrderRepoImpl struct {
	orderDao       dao.IOrderDao
	pageTokenCodec *pagination.CursorCodec
}

func NewOrderRepo(deps OrderRepoDeps) domainrepo.IOrderRepo {
	return &OrderRepoImpl{
		orderDao:       deps.OrderDao,
		pageTokenCodec: deps.PageTokenCodec,
	}
}

func (r *OrderRepoImpl) GetOrder(ctx context.Context, id model.OrderID) (*model.Order, error) {
	if r.orderDao == nil {
		return nil, fmt.Errorf("order dao is nil")
	}

	record, err := r.orderDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order %s: %w", id, err)
	}

	return repomapper.RecordToOrder(record), nil
}

func (r *OrderRepoImpl) ListOrders(ctx context.Context, query model.ListOrderQuery) (model.ListOrderResult, error) {
	if r.orderDao == nil {
		return model.ListOrderResult{}, fmt.Errorf("order dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "order", query.Filter)
	if err != nil {
		return model.ListOrderResult{}, core.NewBadRequestError("resolve order pagination: %v", err)
	}

	records, err := r.orderDao.List(ctx, query.Filter, model.OrderID(afterID), pageSize+1)
	if err != nil {
		return model.ListOrderResult{}, fmt.Errorf("list orders: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.orderDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListOrderResult{}, fmt.Errorf("count orders: %w", countErr)
		}
		totalCount = int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListOrderResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListOrderResult{
		Items:         repomapper.RecordsToOrders(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *OrderRepoImpl) CreateOrder(ctx context.Context, entity *model.Order) (*model.Order, error) {
	if entity == nil {
		return nil, fmt.Errorf("order is nil")
	}

	if r.orderDao == nil {
		return nil, fmt.Errorf("order dao is nil")
	}

	record := repomapper.OrderToRecord(entity)
	if _, err := r.orderDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	return repomapper.RecordToOrder(record), nil
}

func (r *OrderRepoImpl) UpdateOrder(ctx context.Context, entity *model.Order, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("order is nil")
	}

	if r.orderDao == nil {
		return fmt.Errorf("order dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.orderDao.Update(ctx, snapshot.ID, repomapper.OrderFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update order %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *OrderRepoImpl) DeleteOrder(ctx context.Context, id model.OrderID) error {
	if r.orderDao == nil {
		return fmt.Errorf("order dao is nil")
	}

	if err := r.orderDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete order %s: %w", id, err)
	}

	return nil
}
