package app_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gokitmetrics "github.com/fino-io/gokit/metrics"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/api/endpoint"
	middleware "github.com/fino-io/orderpay/backend/modules/orderpay-server/api/endpoint/middleware"
	grpctransport "github.com/fino-io/orderpay/backend/modules/orderpay-server/api/grpc"
	httptransport "github.com/fino-io/orderpay/backend/modules/orderpay-server/api/http"
	api "github.com/fino-io/orderpay/go/orderpay/v1"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestGeneratedHTTPAndGRPC(t *testing.T) {
	service, _, _ := fixture(t)
	tracer := noop.NewTracerProvider().Tracer("acceptance")
	instrumentation := gokitmetrics.Disabled()
	endpoints := middleware.WrapEndpoints(endpoint.NewEndpoints(service), middleware.DefaultOptions(tracer))
	handler := httptransport.NewHandler(endpoints, tracer, instrumentation.HTTPMiddleware)
	request := httptest.NewRequest(http.MethodPost, "/orderpay/v1/orders", strings.NewReader(`{"merchantOrderNo":"http-invalid","amountMinor":-1,"description":"test"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+merchantKey)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())

	listener := bufconn.Listen(1 << 20)
	server := grpctransport.NewServer(endpoints, tracer, instrumentation.GRPCUnaryInterceptor())
	t.Cleanup(server.Stop)
	go func() { _ = server.Serve(listener) }()
	connection, err := grpc.NewClient("passthrough:///local", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	client := api.NewOrderpayServiceClient(connection)
	_, err = client.ListOrders(context.Background(), &api.ListOrdersRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-session-key", merchantKey))
	created, err := client.CreateOrder(ctx, createRequest("grpc-order"))
	require.NoError(t, err)
	_, err = client.CreatePayment(ctx, &api.CreatePaymentRequest{OrderId: created.Order.Id, Payment: &api.PaymentInput{MandateReference: "disabled"}})
	require.Equal(t, codes.Unavailable, status.Code(err))
	conflicting := createRequest("grpc-order")
	conflicting.Order.AmountMinor++
	_, err = client.CreateOrder(ctx, conflicting)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	detail, err := client.GetOrder(ctx, &api.GetOrderRequest{OrderId: created.Order.Id})
	require.NoError(t, err)
	require.Equal(t, created.Order.Id, detail.Order.Id)
}
