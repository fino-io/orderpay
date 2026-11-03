package repo

import (
	"context"

	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
)

//go:generate mockgen -destination=mocks/audit_event.go -package=mocks . IAuditEventRepo
type IAuditEventRepo interface {
	GetAuditEvent(ctx context.Context, id model.AuditEventID) (*model.AuditEvent, error)
	ListAuditEvents(ctx context.Context, query model.ListAuditEventQuery) (model.ListAuditEventResult, error)
	CreateAuditEvent(ctx context.Context, entity *model.AuditEvent) (*model.AuditEvent, error)
	UpdateAuditEvent(ctx context.Context, entity *model.AuditEvent, updateMask []string) error
	DeleteAuditEvent(ctx context.Context, id model.AuditEventID) error
}
