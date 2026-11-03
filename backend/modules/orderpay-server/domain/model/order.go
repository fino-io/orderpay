package model

import (
	"github.com/fino-io/core/go/fino/core"
	pb "github.com/fino-io/orderpay/go/orderpay"
)

type OrderFilter struct {
	IDs    []OrderID
	Status int32
}

func (o *Order) Cancel() error {
	switch o.status {
	case pb.OrderStatus_ORDER_STATUS_CANCELLED:
		return nil
	case pb.OrderStatus_ORDER_STATUS_PENDING_PAYMENT:
		o.status = pb.OrderStatus_ORDER_STATUS_CANCELLED
		o.updateTime = core.Now()
		return nil
	default:
		return ErrConflict
	}
}
