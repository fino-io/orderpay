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

var _ domainrepo.IPaymentRepo = (*MemoryPaymentRepo)(nil)

type MemoryPaymentRepo struct {
	mu             sync.RWMutex
	items          map[model.PaymentID]model.PaymentSnapshot
	pageTokenCodec *pagination.CursorCodec
}

func NewMemoryPaymentRepo(codec *pagination.CursorCodec) domainrepo.IPaymentRepo {
	return &MemoryPaymentRepo{
		items:          make(map[model.PaymentID]model.PaymentSnapshot),
		pageTokenCodec: codec,
	}
}

func (r *MemoryPaymentRepo) GetPayment(ctx context.Context, id model.PaymentID) (*model.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.items[id]
	if !ok {
		return nil, core.NewNotFoundError("payment %s not found", id)
	}

	return model.RebuildPayment(snapshot), nil
}

func (r *MemoryPaymentRepo) ListPayments(ctx context.Context, query model.ListPaymentQuery) (model.ListPaymentResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ids map[model.PaymentID]struct{}
	if len(query.Filter.IDs) > 0 {
		ids = make(map[model.PaymentID]struct{}, len(query.Filter.IDs))
		for _, id := range query.Filter.IDs {
			ids[id] = struct{}{}
		}
	}

	snapshots := make([]model.PaymentSnapshot, 0, len(r.items))
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

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "payment", query.Filter)
	if err != nil {
		return model.ListPaymentResult{}, core.NewBadRequestError("resolve payment pagination: %v", err)
	}

	start := 0
	if afterID != "" {
		start = sort.Search(len(snapshots), func(i int) bool {
			return snapshots[i].ID < model.PaymentID(afterID)
		})
	}
	if start >= len(snapshots) {
		return model.ListPaymentResult{TotalCount: totalCount}, nil
	}

	end := start + pageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	nextPageToken := ""
	if end < len(snapshots) {
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(snapshots[end-1].ID), binding)
		if err != nil {
			return model.ListPaymentResult{}, err
		}
	}
	items := make([]*model.Payment, 0, end-start)
	for _, snapshot := range snapshots[start:end] {
		items = append(items, model.RebuildPayment(snapshot))
	}

	return model.ListPaymentResult{
		Items:         items,
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *MemoryPaymentRepo) CreatePayment(ctx context.Context, entity *model.Payment) (*model.Payment, error) {
	if entity == nil {
		return nil, fmt.Errorf("payment is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return nil, fmt.Errorf("payment id is empty")
	}
	if _, exists := r.items[snapshot.ID]; exists {
		return nil, core.NewAlreadyExistsError("payment %s already exists", snapshot.ID)
	}

	r.items[snapshot.ID] = snapshot

	return model.RebuildPayment(snapshot), nil
}

func (r *MemoryPaymentRepo) UpdatePayment(ctx context.Context, entity *model.Payment, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("payment is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := entity.Snapshot()
	if snapshot.ID == "" {
		return fmt.Errorf("payment id is empty")
	}

	stored, ok := r.items[snapshot.ID]
	if !ok {
		return core.NewNotFoundError("payment %s not found", snapshot.ID)
	}

	updated := model.RebuildPayment(stored)
	if err := updated.ApplyUpdate(model.UpdatePaymentInput{
		UpdateMask:        updateMask,
		MerchantId:        snapshot.MerchantId,
		OrderId:           snapshot.OrderId,
		AmountMinor:       snapshot.AmountMinor,
		MandateReference:  snapshot.MandateReference,
		ChannelAccountId:  snapshot.ChannelAccountId,
		Status:            snapshot.Status,
		ProviderReference: snapshot.ProviderReference,
		ReasonCode:        snapshot.ReasonCode,
		SubmittedAt:       snapshot.SubmittedAt,
		RetryDeadlineAt:   snapshot.RetryDeadlineAt,
		SubmitAttempts:    snapshot.SubmitAttempts,
		NextCheckAt:       snapshot.NextCheckAt,
		CheckAttempts:     snapshot.CheckAttempts,
		CreateTime:        snapshot.CreateTime,
		UpdateTime:        snapshot.UpdateTime,
	}); err != nil {
		return err
	}
	r.items[snapshot.ID] = updated.Snapshot()

	return nil
}

func (r *MemoryPaymentRepo) DeletePayment(ctx context.Context, id model.PaymentID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return core.NewNotFoundError("payment %s not found", id)
	}
	delete(r.items, id)

	return nil
}
