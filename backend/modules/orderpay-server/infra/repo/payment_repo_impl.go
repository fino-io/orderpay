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

var _ domainrepo.IPaymentRepo = (*PaymentRepoImpl)(nil)

type PaymentRepoDeps struct {
	PaymentDao     dao.IPaymentDao
	PageTokenCodec *pagination.CursorCodec
}

type PaymentRepoImpl struct {
	paymentDao     dao.IPaymentDao
	pageTokenCodec *pagination.CursorCodec
}

func NewPaymentRepo(deps PaymentRepoDeps) domainrepo.IPaymentRepo {
	return &PaymentRepoImpl{
		paymentDao:     deps.PaymentDao,
		pageTokenCodec: deps.PageTokenCodec,
	}
}

func (r *PaymentRepoImpl) GetPayment(ctx context.Context, id model.PaymentID) (*model.Payment, error) {
	if r.paymentDao == nil {
		return nil, fmt.Errorf("payment dao is nil")
	}

	record, err := r.paymentDao.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get payment %s: %w", id, err)
	}

	return repomapper.RecordToPayment(record), nil
}

func (r *PaymentRepoImpl) ListPayments(ctx context.Context, query model.ListPaymentQuery) (model.ListPaymentResult, error) {
	if r.paymentDao == nil {
		return model.ListPaymentResult{}, fmt.Errorf("payment dao is nil")
	}

	afterID, pageSize, binding, err := pagination.ResolveKeyset(r.pageTokenCodec, query.PageToken, query.PageSize, "payment", query.Filter)
	if err != nil {
		return model.ListPaymentResult{}, core.NewBadRequestError("resolve payment pagination: %v", err)
	}

	records, err := r.paymentDao.List(ctx, query.Filter, model.PaymentID(afterID), pageSize+1)
	if err != nil {
		return model.ListPaymentResult{}, fmt.Errorf("list payments: %w", err)
	}

	var totalCount int32
	if !query.SkipTotalCount {
		total, countErr := r.paymentDao.Count(ctx, query.Filter)
		if countErr != nil {
			return model.ListPaymentResult{}, fmt.Errorf("count payments: %w", countErr)
		}
		totalCount = int32(total)
	}

	nextPageToken := ""
	if len(records) > pageSize {
		records = records[:pageSize]
		nextPageToken, err = r.pageTokenCodec.EncodeKeyset(string(records[len(records)-1].ID), binding)
		if err != nil {
			return model.ListPaymentResult{}, fmt.Errorf("failed to encode page token: %w", err)
		}
	}

	return model.ListPaymentResult{
		Items:         repomapper.RecordsToPayments(records),
		TotalCount:    totalCount,
		NextPageToken: nextPageToken,
	}, nil
}

func (r *PaymentRepoImpl) CreatePayment(ctx context.Context, entity *model.Payment) (*model.Payment, error) {
	if entity == nil {
		return nil, fmt.Errorf("payment is nil")
	}

	if r.paymentDao == nil {
		return nil, fmt.Errorf("payment dao is nil")
	}

	record := repomapper.PaymentToRecord(entity)
	if _, err := r.paymentDao.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	return repomapper.RecordToPayment(record), nil
}

func (r *PaymentRepoImpl) UpdatePayment(ctx context.Context, entity *model.Payment, updateMask []string) error {
	if entity == nil {
		return fmt.Errorf("payment is nil")
	}

	if r.paymentDao == nil {
		return fmt.Errorf("payment dao is nil")
	}

	snapshot := entity.Snapshot()
	if err := r.paymentDao.Update(ctx, snapshot.ID, repomapper.PaymentFields(snapshot, updateMask)); err != nil {
		return fmt.Errorf("update payment %s: %w", snapshot.ID, err)
	}

	return nil
}

func (r *PaymentRepoImpl) DeletePayment(ctx context.Context, id model.PaymentID) error {
	if r.paymentDao == nil {
		return fmt.Errorf("payment dao is nil")
	}

	if err := r.paymentDao.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete payment %s: %w", id, err)
	}

	return nil
}
