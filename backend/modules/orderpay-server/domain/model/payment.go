package model

import (
	"github.com/fino-io/core/go/fino/core"
	pb "github.com/fino-io/orderpay/go/orderpay"
)

type PaymentFilter struct {
	IDs []PaymentID
}

func (p *Payment) Cancel() error {
	switch p.reasonCode {
	case "CHANNEL_RESULT_CONFLICT", "COMPENSATION_REQUIRED", "RECONCILIATION_REQUIRED":
		return ErrConflict
	}

	switch p.status {
	case pb.PaymentStatus_PAYMENT_STATUS_DECLINED:
		return nil
	case pb.PaymentStatus_PAYMENT_STATUS_CREATED:
		if p.submitAttempts != 0 || p.submittedAt != nil {
			return ErrConflict
		}
		p.status = pb.PaymentStatus_PAYMENT_STATUS_DECLINED
		p.reasonCode = "ORDER_CANCELLED"
		p.nextCheckAt = nil
		p.updateTime = core.Now()
		return nil
	default:
		return ErrConflict
	}
}
