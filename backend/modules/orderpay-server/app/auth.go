package app

import (
	"context"

	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/gokit/middleware/requestid"
	"github.com/fino-io/orderpay/backend/modules/orderpay-server/domain/model"
	"github.com/fino-io/orderpay/backend/pkg/errcode"
	"github.com/google/uuid"
)

type Authenticator interface {
	Authenticate(context.Context) (model.Principal, bool)
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

	if s.readLimiter == nil {
		return ctx, errcode.NewPlatformServiceUnavailable()
	}

	key := principal.MerchantID
	if admin {
		key = "admin"
	}
	allowed, err := s.readLimiter.Allow(ctx, key)
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
