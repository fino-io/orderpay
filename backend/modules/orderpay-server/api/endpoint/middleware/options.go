package middleware

import (
	localemw "github.com/fino-io/gokit/middleware/locale"
	sessionmw "github.com/fino-io/gokit/middleware/session"
	"github.com/fino-io/gokit/session"
	"go.opentelemetry.io/otel/trace"
)

// Options carries transport-neutral endpoint middleware settings.
type Options struct {
	Tracer              trace.Tracer
	EnableValidation    bool
	LocaleResolver      localemw.Resolver
	SessionValidator    session.Validator
	SessionUserResolver sessionmw.UserResolver
}

// DefaultOptions returns the middleware defaults used by this service.
func DefaultOptions(tracer trace.Tracer) Options {
	return Options{
		Tracer:           tracer,
		EnableValidation: true,
	}
}
