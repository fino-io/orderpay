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

var _ domainrepo.IAuditEventRepo = (*AuditEventRepoImpl)(nil)

type AuditEventRepoDeps struct {
	AuditEventDao  dao.IAuditEventDao
	PageTokenCodec *pagination.CursorCodec
}

type AuditEventRepoImpl struct {
	auditEventDao  dao.IAuditEventDao
	pageTokenCodec *pagination.CursorCodec
}

func NewAuditEventRepo(deps AuditEventRepoDeps) domainrepo.IAuditEventRepo {
	return &AuditEventRepoImpl{
		auditEventDao:  deps.AuditEventDao,
		pageTokenCodec: deps.PageTokenCodec,
	}
}

func (r *AuditEventRepoImpl) GetAuditEvent(ctx context.Context, id model.AuditEventID) (*model.AuditEvent, error) {
	if r.auditEventDao == nil {
		return nil, fmt.Errorf("audit_event dao is nil")
	}

	record, err := r.auditEventDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get audit_event %s: %w", id, err)
	}

	return repomapper.RecordToAuditEvent(record), nil
}

func (r *AuditEventRepoImpl) ListAuditEvents(ctx context.Context, query model.ListAuditEventQuery) (model.ListAuditEventResult, error) {
	if r.auditEventDao == nil {
		return model.ListAuditEventResult{}, fmt.Errorf("audit_event dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "audit_event", query.Filter)
	if err != nil {
		return model.ListAuditEventResult{}, core.NewBadRequestError("resolve audit_event pagination: %v", err)
	}

	records, err := r.auditEventDao.List(ctx, query.Filter, model.AuditEventID(afterID), pageSize+1)
	if err != nil {
		return model.ListAuditEventResult{}, fmt.Errorf("list audit_events: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.auditEventDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListAuditEventResult{}, fmt.Errorf("count audit_events: %w", countErr)
		}
		totalCount = int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListAuditEventResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListAuditEventResult{
		Items:         repomapper.RecordsToAuditEvents(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *AuditEventRepoImpl) CreateAuditEvent(ctx context.Context, entity *model.AuditEvent) (*model.AuditEvent, error) {
	if entity == nil {
		return nil, fmt.Errorf("audit_event is nil")
	}

	if r.auditEventDao == nil {
		return nil, fmt.Errorf("audit_event dao is nil")
	}

	record := repomapper.AuditEventToRecord(entity)
	if _, err := r.auditEventDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create audit_event: %w", err)
	}

	return repomapper.RecordToAuditEvent(record), nil
}

func (r *AuditEventRepoImpl) UpdateAuditEvent(ctx context.Context, entity *model.AuditEvent, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("audit_event is nil")
	}

	if r.auditEventDao == nil {
		return fmt.Errorf("audit_event dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.auditEventDao.Update(ctx, snapshot.ID, repomapper.AuditEventFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update audit_event %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *AuditEventRepoImpl) DeleteAuditEvent(ctx context.Context, id model.AuditEventID) error {
	if r.auditEventDao == nil {
		return fmt.Errorf("audit_event dao is nil")
	}

	if err := r.auditEventDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete audit_event %s: %w", id, err)
	}

	return nil
}
