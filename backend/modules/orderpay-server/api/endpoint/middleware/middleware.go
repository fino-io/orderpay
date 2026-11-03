package middleware

import (
	svcendpoint "github.com/fino-io/orderpay/backend/modules/orderpay-server/api/endpoint"
)

// WrapEndpoints composes endpoint middleware in execution order:
// tracing -> context errors -> locale -> session -> rate limit -> context cache -> validation -> endpoint.
func WrapEndpoints(in svcendpoint.Endpoints, options Options) svcendpoint.Endpoints {
	if middleware := validatorMiddleware(options); middleware != nil {
		in.WrapAll(middleware)
	}
	wrapContextCachedEndpoints(&in)
	wrapRateLimitedEndpoints(&in)
	if middleware := sessionMiddleware(options); middleware != nil {
		in.WrapAll(middleware)
	}
	if middleware := localeMiddleware(options); middleware != nil {
		in.WrapAll(middleware)
	}
	in.WrapAllLabeled(contextErrorMiddleware)
	if middleware := tracingMiddleware(options); middleware != nil {
		in.WrapAllLabeled(middleware)
	}

	return in
}

// wrapContextCachedEndpoints configures endpoints that need request-scoped caching.
func wrapContextCachedEndpoints(_ *svcendpoint.Endpoints) {
}

// wrapRateLimitedEndpoints configures endpoints protected by the local rate limiter.
func wrapRateLimitedEndpoints(in *svcendpoint.Endpoints) {
	if middleware := rateLimitMiddleware(); middleware != nil {
		in.WrapAll(middleware)
	}
}
