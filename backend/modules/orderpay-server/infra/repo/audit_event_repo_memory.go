package repo

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/pagination"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	domainrepo "github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/repo"
)

var _ domainrepo.IAuditEventRepo = (*MemoryAuditEventRepo)(nil)

type MemoryAuditEventRepo struct {
	mu             sync.RWMutex
	items          map[model.AuditEventID]model.AuditEventSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemoryAuditEventRepo(codec *pagination.CursorCodec) domainrepo.IAuditEventRepo {
	return &MemoryAuditEventRepo{
		items:          make(map[model.AuditEventID]model.AuditEventSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemoryAuditEventRepo) GetAuditEvent(ctx context.Context, id model.AuditEventID) (*model.AuditEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("audit_event %s not found", id)
	}

	return model.RebuildAuditEvent(snapshot), nil
}

func (r *MemoryAuditEventRepo) ListAuditEvents(ctx context.Context, query model.ListAuditEventQuery) (model.ListAuditEventResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.AuditEventID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.AuditEventID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.AuditEventSnapshot, 0, len(r.items))
	for id, snapshot := range r.items {
		if ids != nil {
			if _, ok := ids[id]; !ok {
				continue
			}
		}
		snapshots = append(snapshots, snapshot)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].ID > snapshots[j].ID
	})

	var totalCount int32
	if !query.SkipTotalCount {
		totalCount = int32(len(snapshots))
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "audit_event", query.Filter)
	if err != nil {
		return model.ListAuditEventResult{}, core.NewBadRequestError("resolve audit_event pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.AuditEventID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListAuditEventResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListAuditEventResult{}, err
		}
	}
	items := make([]*model.AuditEvent, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildAuditEvent(snapshot))
	}

	return model.ListAuditEventResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemoryAuditEventRepo) CreateAuditEvent(ctx context.Context, entity *model.AuditEvent) (*model.AuditEvent, error) {
	if entity == nil {
		return nil, fmt.Errorf("audit_event is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("audit_event id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("audit_event %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildAuditEvent(snapshot), nil
}

func (r *MemoryAuditEventRepo) UpdateAuditEvent(ctx context.Context, entity *model.AuditEvent, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("audit_event is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("audit_event id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("audit_event %s not found", snapshot.ID)
	}

	updated := model.RebuildAuditEvent(stored)
	if err := updated.ApplyUpdate(model.UpdateAuditEventInput{
		UpdateMask:        updateMask,
		MerchantId:        snapshot.MerchantId,
		ActorId:           snapshot.ActorId,
		Action:            snapshot.Action,
		ResourceId:        snapshot.ResourceId,
		RequestId:         snapshot.RequestId,
		BeforeStatus:      snapshot.BeforeStatus,
		AfterStatus:       snapshot.AfterStatus,
		ReasonCode:        snapshot.ReasonCode,
		EvidenceReference: snapshot.EvidenceReference,
		CreateTime:        snapshot.CreateTime,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemoryAuditEventRepo) DeleteAuditEvent(ctx context.Context, id model.AuditEventID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("audit_event %s not found", id)
	}
	delete(r.items, id)

	return nil
}
