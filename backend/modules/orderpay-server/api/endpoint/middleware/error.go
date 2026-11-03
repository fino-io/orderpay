package middleware

import (
	"context"
	"errors"

	"github.com/fino-io/finokit/logs"
	"github.com/go-kit/kit/endpoint"
)

// contextErrorMiddleware logs interrupted requests without changing their error chain.
// HTTP and gRPC handle the protocol-specific error conversion.
func contextErrorMiddleware(name string, next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		response, err := next(ctx, request)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logs.Ctx(ctx).Warnw(name+" interrupted", "reason", "context_done")
		}
		return response, err
	}
}
