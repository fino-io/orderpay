package app_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/pagination"
	"github.com/fino-io/gokit/middleware/ratelimit"
	"github.com/fino-io/gokit/session"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/app"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	domainrepo "github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/repo"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/service"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/auth"
	repository "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo"
	record "github.com/fino-io/orderpay/backend/modules/orderpay-server/infra/repo/model/sqlite"
	pb "github.com/fino-io/orderpay/go/orderpay"
	api "github.com/fino-io/orderpay/go/orderpay/v1"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const merchantID = "019a4a70-0000-7000-8000-000000000001"
const otherID = "019a4a70-0000-7000-8000-000000000002"
const merchantKey = "merchant-secret-key-for-local-test-only"
const adminKey = "administrator-secret-for-local-test-only"

func fixture(t *testing.T, verifiers ...service.MandateVerifier) (api.OrderpayServiceServer, *service.Workflow, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "orders.db")+"?_journal_mode=WAL&_synchronous=FULL&_foreign_keys=on&_busy_timeout=1000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := database.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, database.AutoMigrate(&record.Order{}, &record.Payment{}, &record.AuditEvent{}))
	codec, err := pagination.NewCursorCodec(make([]byte, 32))
	require.NoError(t, err)
	store := repository.NewWorkflowStore(database, codec)
	require.NoError(t, store.Initialize())
	var verifier service.MandateVerifier
	if len(verifiers) > 0 {
		verifier = verifiers[0]
	}

	credentials := auth.Credentials{
		sha256.Sum256([]byte(merchantKey)): {
			MerchantID: merchantID,
			ActorID:    merchantID,
		},
		sha256.Sum256([]byte(adminKey)): {
			Admin:   true,
			ActorID: "admin",
		},
		sha256.Sum256([]byte(merchantKey + "other")): {
			MerchantID: otherID,
			ActorID:    otherID,
		},
	}
	readLimiter, err := ratelimit.NewLocalKeyedRateLimiter(time.Millisecond, 1000, 100, time.Minute)
	require.NoError(t, err)
	writeLimiter, err := ratelimit.NewLocalKeyedRateLimiter(time.Millisecond, 1000, 100, time.Minute)
	require.NoError(t, err)
	workflow := service.NewWorkflow(store, verifier, writeLimiter)
	return app.NewService(app.ServiceDeps{
		Workflow:    workflow,
		Credentials: credentials,
		ReadLimiter: readLimiter,
	}), workflow, database
}

func merchantContext() context.Context {
	return session.WithToken(context.Background(), merchantKey)
}

func createRequest(number string) *api.CreateOrderRequest {
	return &api.CreateOrderRequest{
		Order: &api.OrderInput{
			MerchantOrderNo: number,
			AmountMinor:     1990,
			Description:     "test",
		},
	}
}

func TestOrderWorkflow(t *testing.T) {
	server, _, database := fixture(t)
	ctx := merchantContext()
	first, err := server.CreateOrder(ctx, createRequest("order-1"))
	require.NoError(t, err)
	replay, err := server.CreateOrder(ctx, createRequest("order-1"))
	require.NoError(t, err)
	require.Equal(t, first.Order.Id, replay.Order.Id)
	changed := createRequest("order-1")
	changed.Order.AmountMinor++
	_, err = server.CreateOrder(ctx, changed)
	require.Error(t, err)
	detail, err := server.GetOrder(ctx, &api.GetOrderRequest{OrderId: first.Order.Id})
	require.NoError(t, err)
	require.Nil(t, detail.Payment)
	cancelled, err := server.CancelOrder(ctx, &api.CancelOrderRequest{OrderId: first.Order.Id})
	require.NoError(t, err)
	require.Equal(t, pb.OrderStatus_ORDER_STATUS_CANCELLED, cancelled.Order.Status)
	_, err = server.CancelOrder(ctx, &api.CancelOrderRequest{OrderId: first.Order.Id})
	require.NoError(t, err)
	var events []record.AuditEvent
	require.NoError(t, database.Find(&events).Error)
	require.Len(t, events, 2)
	require.NotEmpty(t, events[0].RequestId)
	require.Error(t, database.Exec("DELETE FROM audit_event").Error)
	require.Error(t, database.Exec("UPDATE audit_event SET reason_code='changed'").Error)
	require.Error(t, database.Exec("UPDATE `order` SET amount_minor=1").Error)
}

func TestIsolationAndAuditAuthorization(t *testing.T) {
	server, _, _ := fixture(t)
	first, err := server.CreateOrder(merchantContext(), createRequest("private"))
	require.NoError(t, err)
	for _, ctx := range []context.Context{context.Background(), session.WithToken(context.Background(), merchantKey+"other")} {
		_, err := server.GetOrder(ctx, &api.GetOrderRequest{OrderId: first.Order.Id})
		require.Error(t, err)
	}
	_, err = server.ListAuditEvents(merchantContext(), &api.ListAuditEventsRequest{})
	require.Error(t, err)
	audit, err := server.ListAuditEvents(session.WithToken(context.Background(), adminKey), &api.ListAuditEventsRequest{MerchantId: merchantID})
	require.NoError(t, err)
	require.Len(t, audit.Events, 1)
	require.Equal(t, "ORDER_CREATED", audit.Events[0].Action)
}

func TestPaginationScope(t *testing.T) {
	server, _, _ := fixture(t)
	for _, key := range []string{"one", "two", "three"} {
		_, err := server.CreateOrder(merchantContext(), createRequest(key))
		require.NoError(t, err)
	}
	first, err := server.ListOrders(merchantContext(), &api.ListOrdersRequest{PageSize: 2})
	require.NoError(t, err)
	require.Len(t, first.Orders, 2)
	require.NotEmpty(t, first.NextPageToken)
	next, err := server.ListOrders(merchantContext(), &api.ListOrdersRequest{
		PageSize:  2,
		PageToken: first.NextPageToken,
	})
	require.NoError(t, err)
	require.Len(t, next.Orders, 1)
	require.NotEqual(t, first.Orders[0].Id, next.Orders[0].Id)
	_, err = server.ListOrders(session.WithToken(context.Background(), merchantKey+"other"), &api.ListOrdersRequest{
		PageSize:  2,
		PageToken: first.NextPageToken,
	})
	require.Error(t, err)
}

func TestConcurrentCreateAndAuditRollback(t *testing.T) {
	server, _, database := fixture(t)
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := server.CreateOrder(merchantContext(), createRequest("concurrent"))
			if err != nil {
				t.Error(err)
				return
			}
			ids <- response.Order.Id
		}()
	}
	wg.Wait()
	close(ids)
	expected := ""
	for id := range ids {
		if expected == "" {
			expected = id
		}
		require.Equal(t, expected, id)
	}
	require.NoError(t, database.Exec(`CREATE TRIGGER reject_audit BEFORE INSERT ON audit_event BEGIN SELECT RAISE(ABORT, 'test audit failure'); END`).Error)
	_, err := server.CreateOrder(merchantContext(), createRequest("rollback"))
	require.Error(t, err)
	var count int64
	require.NoError(t, database.Model(&record.Order{}).Where("merchant_order_no = ?", "rollback").Count(&count).Error)
	require.Zero(t, count)
}

func TestChannelDisabled(t *testing.T) {
	server, workflow, database := fixture(t)
	first, err := server.CreateOrder(merchantContext(), createRequest("no-channel"))
	require.NoError(t, err)
	ctx := model.WithPrincipal(context.Background(), model.Principal{
		MerchantID: merchantID,
		ActorID:    merchantID,
	})
	_, err = workflow.CreatePayment(ctx, first.Order.Id, "opaque-reference")
	require.True(t, errors.Is(err, model.ErrUnavailable))
	var count int64
	require.NoError(t, database.Model(&record.Payment{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestCancelUnsubmittedAndRejectPending(t *testing.T) {
	for _, status := range []pb.PaymentStatus{pb.PaymentStatus_PAYMENT_STATUS_CREATED, pb.PaymentStatus_PAYMENT_STATUS_PENDING} {
		t.Run(status.String(), func(t *testing.T) {
			server, _, database := fixture(t)
			order, err := server.CreateOrder(merchantContext(), createRequest("payment"))
			require.NoError(t, err)
			payment := record.Payment{
				ID:               model.PaymentID("019a4a70-0000-7000-8000-000000000003"),
				MerchantId:       merchantID,
				OrderId:          order.Order.Id,
				AmountMinor:      1990,
				MandateReference: "test",
				ChannelAccountId: "disabled-test-fixture",
				Status:           status,
				CreateTime:       core.Now(),
				UpdateTime:       core.Now(),
			}
			require.NoError(t, database.Create(&payment).Error)
			_, err = server.CancelOrder(merchantContext(), &api.CancelOrderRequest{OrderId: order.Order.Id})
			if status == pb.PaymentStatus_PAYMENT_STATUS_PENDING {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NoError(t, database.First(&payment, "id = ?", payment.ID).Error)
			require.Equal(t, pb.PaymentStatus_PAYMENT_STATUS_DECLINED, payment.Status)
			require.Equal(t, "ORDER_CANCELLED", payment.ReasonCode)
		})
	}
}

type mandateVerifierFunc func(context.Context, string, string) (string, error)

func (f mandateVerifierFunc) Authorize(ctx context.Context, merchant, reference string) (string, error) {
	return f(ctx, merchant, reference)
}

func TestPaymentAcceptanceWithVerifiedTestAuthorization(t *testing.T) {
	verifier := mandateVerifierFunc(func(_ context.Context, merchant, reference string) (string, error) {
		require.Equal(t, merchantID, merchant)
		require.Equal(t, "test-mandate", reference)
		return "test-account", nil
	})
	server, _, database := fixture(t, verifier)
	order, err := server.CreateOrder(merchantContext(), createRequest("acceptance"))
	require.NoError(t, err)
	request := &api.CreatePaymentRequest{
		OrderId: order.Order.Id,
		Payment: &api.PaymentInput{MandateReference: "test-mandate"},
	}
	first, err := server.CreatePayment(merchantContext(), request)
	require.NoError(t, err)
	require.Equal(t, int64(1990), first.Payment.AmountMinor)
	require.Equal(t, pb.PaymentStatus_PAYMENT_STATUS_CREATED, first.Payment.Status)
	replay, err := server.CreatePayment(merchantContext(), request)
	require.NoError(t, err)
	require.Equal(t, first.Payment.Id, replay.Payment.Id)
	request.Payment.MandateReference = "different-mandate"
	_, err = server.CreatePayment(merchantContext(), request)
	require.Error(t, err)
	_, err = server.CancelOrder(merchantContext(), &api.CancelOrderRequest{OrderId: order.Order.Id})
	require.NoError(t, err)
	var count int64
	require.NoError(t, database.Model(&record.Payment{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

type deniedWriteLimiter struct{}

func (deniedWriteLimiter) Allow(context.Context, string) (bool, error) {
	return false, nil
}

func TestWriteQuotaAllowsExistingOrderReplay(t *testing.T) {
	server, _, database := fixture(t)
	request := createRequest("existing")
	existing, err := server.CreateOrder(merchantContext(), request)
	require.NoError(t, err)
	codec, err := pagination.NewCursorCodec(make([]byte, 32))
	require.NoError(t, err)
	workflow := service.NewWorkflow(repository.NewWorkflowStore(database, codec), nil, deniedWriteLimiter{})
	ctx := model.WithPrincipal(context.Background(), model.Principal{
		MerchantID: merchantID,
		ActorID:    merchantID,
	})
	replay, err := workflow.CreateOrder(ctx, model.CreateOrderInput{
		MerchantOrderNo: "existing",
		AmountMinor:     1990,
		Description:     "test",
	})
	require.NoError(t, err)
	require.Equal(t, existing.Order.Id, string(replay.ID()))
	_, err = workflow.CreateOrder(ctx, model.CreateOrderInput{
		MerchantOrderNo: "new",
		AmountMinor:     1990,
		Description:     "test",
	})
	require.ErrorIs(t, err, model.ErrRateLimited)
}

func TestCancellationDoesNotLeaveSQLiteTransactionOpen(t *testing.T) {
	server, _, database := fixture(t)
	codec, err := pagination.NewCursorCodec(make([]byte, 32))
	require.NoError(t, err)
	store := repository.NewWorkflowStore(database, codec)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = store.Transaction(ctx, func(domainrepo.WorkflowTx) error {
		cancel()
		return ctx.Err()
	})
	require.Error(t, err)
	_, err = server.CreateOrder(merchantContext(), createRequest("after-cancellation"))
	require.NoError(t, err)
}
