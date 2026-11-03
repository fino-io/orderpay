package repo

import (
	"context"

	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
)

//go:generate mockgen -destination=mocks/order.go -package=mocks . IOrderRepo
type IOrderRepo interface {
	GetOrder(ctx context.Context, id model.OrderID) (*model.Order, error)
	ListOrders(ctx context.Context, query model.ListOrderQuery) (model.ListOrderResult, error)
	CreateOrder(ctx context.Context, entity *model.Order) (*model.Order, error)
	UpdateOrder(ctx context.Context, entity *model.Order, updateMask []string) error
	DeleteOrder(ctx context.Context, id model.OrderID) error
}
