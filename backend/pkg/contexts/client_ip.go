package contexts

import "context"

type clientIPContextKey struct{}

// WithClientIP stores the transport-level client IP in context.
func WithClientIP(ctx context.Context, ip string) context.Context {
	if ip == "" {
		return ctx
	}
	return context.WithValue(ctx, clientIPContextKey{}, ip)
}

// ClientIP returns the client IP stored by the transport layer.
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPContextKey{}).(string)
	return ip
}
