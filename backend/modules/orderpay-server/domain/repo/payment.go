package repo

import (
	"context"

	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
)

//go:generate mockgen -destination=mocks/payment.go -package=mocks . IPaymentRepo
type IPaymentRepo interface {
	GetPayment(ctx context.Context, id model.PaymentID) (*model.Payment, error)
	ListPayments(ctx context.Context, query model.ListPaymentQuery) (model.ListPaymentResult, error)
	CreatePayment(ctx context.Context, entity *model.Payment) (*model.Payment, error)
	UpdatePayment(ctx context.Context, entity *model.Payment, updateMask []string) error
	DeletePayment(ctx context.Context, id model.PaymentID) error
}
