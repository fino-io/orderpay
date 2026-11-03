package repo

import (
	"context"

	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
)

type WorkflowTx interface {
	Order(merchantID, id string) (*model.Order, error)
	OrderByNumber(merchantID, number string) (*model.Order, error)
	Payment(merchantID, orderID string) (*model.Payment, error)
	SaveOrder(*model.Order) error
	SavePayment(*model.Payment) error
	AppendAudit(*model.AuditEvent) error
}

type WorkflowStore interface {
	Transaction(context.Context, func(WorkflowTx) error) error
	Orders(context.Context, string, model.ListOrderQuery) (model.ListOrderResult, error)
	Audits(context.Context, string, string, int, string) (model.ListAuditEventResult, error)
}
