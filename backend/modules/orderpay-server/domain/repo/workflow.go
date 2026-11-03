package repo

import (
	"context"

	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
)

type WorkflowReader interface {
	Order(merchantID, id string) (*model.Order, error)
	OrderByNumber(merchantID, number string) (*model.Order, error)
	Payment(merchantID, orderID string) (*model.Payment, error)
}

type WorkflowTx interface {
	WorkflowReader
	SaveOrder(*model.Order) error
	SavePayment(*model.Payment) error
	AppendAudit(*model.AuditEvent) error
}

type WorkflowStore interface {
	View(context.Context, func(WorkflowReader) error) error
	Transaction(context.Context, func(WorkflowTx) error) error
	Orders(context.Context, string, model.ListOrderQuery) (model.ListOrderResult, error)
	Audits(context.Context, string, string, int, string) (model.ListAuditEventResult, error)
}
