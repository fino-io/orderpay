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

var _ domainrepo.IOrderRepo = (*MemoryOrderRepo)(nil)

type MemoryOrderRepo struct {
	mu             sync.RWMutex
	items          map[model.OrderID]model.OrderSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemoryOrderRepo(codec *pagination.CursorCodec) domainrepo.IOrderRepo {
	return &MemoryOrderRepo{
		items:          make(map[model.OrderID]model.OrderSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemoryOrderRepo) GetOrder(ctx context.Context, id model.OrderID) (*model.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("order %s not found", id)
	}

	return model.RebuildOrder(snapshot), nil
}

func (r *MemoryOrderRepo) ListOrders(ctx context.Context, query model.ListOrderQuery) (model.ListOrderResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.OrderID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.OrderID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.OrderSnapshot, 0, len(r.items))
	for id, snapshot := range r.items {
		if ids != nil {
			if _, ok := ids[id]; !ok {
				continue
			}
		}
		if query.Filter.Status != 0 && int32(snapshot.Status) != query.Filter.Status {
			continue
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

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "order", query.Filter)
	if err != nil {
		return model.ListOrderResult{}, core.NewBadRequestError("resolve order pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.OrderID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListOrderResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListOrderResult{}, err
		}
	}
	items := make([]*model.Order, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildOrder(snapshot))
	}

	return model.ListOrderResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemoryOrderRepo) CreateOrder(ctx context.Context, entity *model.Order) (*model.Order, error) {
	if entity == nil {
		return nil, fmt.Errorf("order is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("order id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("order %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildOrder(snapshot), nil
}

func (r *MemoryOrderRepo) UpdateOrder(ctx context.Context, entity *model.Order, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("order is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("order id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("order %s not found", snapshot.ID)
	}

	updated := model.RebuildOrder(stored)
	if err := updated.ApplyUpdate(model.UpdateOrderInput{
		UpdateMask:      updateMask,
		MerchantId:      snapshot.MerchantId,
		MerchantOrderNo: snapshot.MerchantOrderNo,
		AmountMinor:     snapshot.AmountMinor,
		Description:     snapshot.Description,
		Status:          snapshot.Status,
		CreateTime:      snapshot.CreateTime,
		UpdateTime:      snapshot.UpdateTime,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemoryOrderRepo) DeleteOrder(ctx context.Context, id model.OrderID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("order %s not found", id)
	}
	delete(r.items, id)

	return nil
}
