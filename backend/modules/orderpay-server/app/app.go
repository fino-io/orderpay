package app

import (
	"context"
	"errors"

	"github.com/fino-io/core/go/fino/core"
	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/gokit/middleware/ratelimit"
	"github.com/fino-io/gokit/middleware/requestid"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/app/mapper"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/service"
	"github.com/fino-io/orderpay/backend/pkg/errcode"
	pb "github.com/fino-io/orderpay/go/orderpay/v1"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context) (model.Principal, bool)
}

type ServiceDeps struct {
	ReadLimiter ratelimit.KeyedLimiter
	Workflow    *service.Workflow
	Credentials Authenticator
}

type orderpayServer struct {
	pb.UnimplementedOrderpayServiceServer

	workflow    *service.Workflow
	credentials Authenticator
	readLimiter ratelimit.KeyedLimiter
}

func NewService(deps ServiceDeps) pb.OrderpayServiceServer {
	return orderpayServer{
		workflow:    deps.Workflow,
		credentials: deps.Credentials,
		readLimiter: deps.ReadLimiter,
	}
}

func (s orderpayServer) authorize(ctx context.Context, admin bool) (context.Context, error) {
	if s.credentials == nil {
		return ctx, errcode.NewPlatformUnauthenticated()
	}
	principal, ok := s.credentials.Authenticate(ctx)
	if !ok {
		logs.Ctx(ctx).Warnw("authorization rejected", "reason", "invalid_credential")
		return ctx, errcode.NewPlatformUnauthenticated()
	}
	if admin != principal.Admin {
		logs.Ctx(ctx).Warnw("authorization rejected", "reason", "role_forbidden")
		return ctx, errcode.NewPlatformForbidden()
	}
	limiter := s.readLimiter
	if limiter == nil {
		return ctx, errcode.NewPlatformServiceUnavailable()
	}
	key := principal.MerchantID
	if admin {
		key = "admin"
	}
	allowed, err := limiter.Allow(ctx, key)
	if err != nil {
		return ctx, errcode.WrapPlatformServiceUnavailable(err)
	}
	if !allowed {
		return ctx, errcode.NewPlatformTooManyRequests()
	}
	principal.RequestID = requestid.FromContext(ctx)
	if principal.RequestID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return ctx, errcode.WrapPlatformInternalServerError(err)
		}
		principal.RequestID = id.String()
	}
	return model.WithPrincipal(ctx, principal), nil
}

// CreateOrder implements Interface. fino:req-fields 1=Order
func (s orderpayServer) CreateOrder(ctx context.Context, in *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	logs.Ctx(ctx).Debugw("CreateOrder")
	if in == nil {
		logs.Ctx(ctx).Warnw("request missing", "reason", "request_missing")
		return nil, errcode.NewOrderpayRequestMissing()
	}

	ctx, err := s.authorize(ctx, false)
	if err != nil {
		return nil, err
	}

	input, err := mapper.ToCreateOrderInput(in.GetOrder())
	if err != nil {
		logs.Ctx(ctx).Warnw("invalid order input", "reason", "input_invalid")
		return nil, errcode.WrapCreateOrderpayInputInvalid(err)
	}

	entity, err := s.workflow.CreateOrder(ctx, input)
	if err != nil {
		if errors.Is(err, model.ErrRateLimited) {
			return nil, errcode.WrapPlatformTooManyRequests(err)
		}
		if errors.Is(err, model.ErrConflict) {
			logs.Ctx(ctx).Warnw("business conflict", "reason", "conflict")
			return nil, errcode.WrapOrderConflict(err)
		}
		if core.IsNotFoundError(err) {
			return nil, errcode.WrapOrderpayNotFound(err)
		}
		if core.IsBadRequestError(err) {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		return nil, errcode.WrapCreateOrderpayFailed(err)
	}

	logs.Ctx(ctx).Infow("operation completed")

	return &pb.CreateOrderResponse{
		Order: mapper.ToOrderView(entity),
	}, nil
}

// GetOrder implements Interface. fino:req-fields 1=OrderId
func (s orderpayServer) GetOrder(ctx context.Context, in *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	logs.Ctx(ctx).Debugw("GetOrder")
	if in == nil {
		logs.Ctx(ctx).Warnw("request missing", "reason", "request_missing")
		return nil, errcode.NewOrderpayRequestMissing()
	}

	ctx, err := s.authorize(ctx, false)
	if err != nil {
		return nil, err
	}

	entity, payment, err := s.workflow.GetOrder(ctx, in.GetOrderId())
	if err != nil {
		if core.IsNotFoundError(err) {
			return nil, errcode.WrapOrderpayNotFound(err)
		}
		if core.IsBadRequestError(err) {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		return nil, errcode.WrapGetOrderpayFailed(err)
	}

	return &pb.GetOrderResponse{
		Order:   mapper.ToOrderView(entity),
		Payment: mapper.ToPaymentView(payment),
	}, nil
}

// ListOrders implements Interface. fino:req-fields 1=Status,90=PageSize,91=PageToken
func (s orderpayServer) ListOrders(ctx context.Context, in *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	logs.Ctx(ctx).Debugw("ListOrders")
	if in == nil {
		logs.Ctx(ctx).Warnw("request missing", "reason", "request_missing")
		return nil, errcode.NewOrderpayRequestMissing()
	}

	ctx, err := s.authorize(ctx, false)
	if err != nil {
		return nil, err
	}

	result, err := s.workflow.ListOrders(ctx, model.ListOrderQuery{
		Filter:         model.OrderFilter{Status: int32(in.GetStatus())},
		PageSize:       int(in.GetPageSize()),
		PageToken:      in.GetPageToken(),
		SkipTotalCount: true,
	})
	if err != nil {
		if core.IsNotFoundError(err) {
			return nil, errcode.WrapOrderpayNotFound(err)
		}
		if core.IsBadRequestError(err) {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		return nil, errcode.WrapListOrderpaysFailed(err)
	}

	return &pb.ListOrdersResponse{
		Orders:        mapper.ToOrderViews(result.Items),
		NextPageToken: result.NextPageToken,
	}, nil
}

// CancelOrder implements Interface. fino:req-fields 1=OrderId
func (s orderpayServer) CancelOrder(ctx context.Context, in *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	logs.Ctx(ctx).Debugw("CancelOrder")
	if in == nil {
		logs.Ctx(ctx).Warnw("request missing", "reason", "request_missing")
		return nil, errcode.NewOrderpayRequestMissing()
	}

	ctx, err := s.authorize(ctx, false)
	if err != nil {
		return nil, err
	}

	entity, err := s.workflow.CancelOrder(ctx, in.GetOrderId())
	if err != nil {
		if errors.Is(err, model.ErrRateLimited) {
			return nil, errcode.WrapPlatformTooManyRequests(err)
		}
		if errors.Is(err, model.ErrConflict) {
			logs.Ctx(ctx).Warnw("business conflict", "reason", "conflict")
			return nil, errcode.WrapOrderConflict(err)
		}
		if core.IsNotFoundError(err) {
			return nil, errcode.WrapOrderpayNotFound(err)
		}
		if core.IsBadRequestError(err) {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		return nil, errcode.WrapCancelOrderFailed(err)
	}

	logs.Ctx(ctx).Infow("operation completed")

	return &pb.CancelOrderResponse{
		Order: mapper.ToOrderView(entity),
	}, nil
}

// CreatePayment implements Interface. fino:req-fields 1=OrderId,2=Payment
func (s orderpayServer) CreatePayment(ctx context.Context, in *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	logs.Ctx(ctx).Debugw("CreatePayment")
	if in == nil {
		logs.Ctx(ctx).Warnw("request missing", "reason", "request_missing")
		return nil, errcode.NewOrderpayRequestMissing()
	}

	ctx, err := s.authorize(ctx, false)
	if err != nil {
		return nil, err
	}

	if in.GetPayment() == nil {
		return nil, errcode.NewCreateOrderpayInputInvalid()
	}
	entity, err := s.workflow.CreatePayment(ctx, in.GetOrderId(), in.GetPayment().GetMandateReference())
	if err != nil {
		if errors.Is(err, model.ErrRateLimited) {
			return nil, errcode.WrapPlatformTooManyRequests(err)
		}
		if errors.Is(err, model.ErrConflict) {
			logs.Ctx(ctx).Warnw("business conflict", "reason", "conflict")
			return nil, errcode.WrapPaymentConflict(err)
		}
		if errors.Is(err, model.ErrUnavailable) {
			logs.Ctx(ctx).Warnw("payment unavailable", "reason", "channel_disabled")
			return nil, errcode.WrapPaymentUnavailable(err)
		}
		if core.IsNotFoundError(err) {
			return nil, errcode.WrapOrderpayNotFound(err)
		}
		if core.IsBadRequestError(err) {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		return nil, errcode.WrapCreatePaymentFailed(err)
	}

	logs.Ctx(ctx).Infow("operation completed")

	return &pb.CreatePaymentResponse{
		Payment: mapper.ToPaymentView(entity),
	}, nil
}

// ListAuditEvents implements Interface. fino:req-fields 1=MerchantId,2=ResourceId,90=PageSize,91=PageToken
func (s orderpayServer) ListAuditEvents(ctx context.Context, in *pb.ListAuditEventsRequest) (*pb.ListAuditEventsResponse, error) {
	logs.Ctx(ctx).Debugw("ListAuditEvents")
	if in == nil {
		logs.Ctx(ctx).Warnw("request missing", "reason", "request_missing")
		return nil, errcode.NewOrderpayRequestMissing()
	}

	ctx, err := s.authorize(ctx, true)
	if err != nil {
		return nil, err
	}

	result, err := s.workflow.ListAudits(ctx, in.GetMerchantId(), in.GetResourceId(), int(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		if core.IsNotFoundError(err) {
			return nil, errcode.WrapOrderpayNotFound(err)
		}
		if core.IsBadRequestError(err) {
			return nil, errcode.WrapPlatformBadRequest(err)
		}
		return nil, errcode.WrapListAuditEventsFailed(err)
	}

	return &pb.ListAuditEventsResponse{
		Events:        mapper.ToAuditEventViews(result.Items),
		NextPageToken: result.NextPageToken,
	}, nil
}
