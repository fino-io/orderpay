package middleware

import (
	"context"
	"time"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
	"github.com/fino-io/gokit/middleware/ratelimit"
	"github.com/fino-io/gokit/session"
	"github.com/fino-io/orderpay/backend/pkg/contexts"
	"github.com/fino-io/orderpay/backend/pkg/errcode"
	goendpoint "github.com/go-kit/kit/endpoint"
)

const rateLimitAnonymousKey = "anonymous"

type rateLimitConfig struct {
	Enabled bool          `json:"enabled"`
	Every   time.Duration `json:"every"`   // 50 * time.Millisecond, 每 50ms 恢复 1 个 token, 即长期平均 20 QPS
	Burst   int           `json:"burst"`   // 40, 桶最多存 40 个 token，即短时间最多允许连续打进来 40 个请求
	MaxKeys int           `json:"maxKeys"` // 当前进程最多保留的用户/IP bucket 数量
	TTL     time.Duration `json:"ttl"`     // 10 * time.Minute, 后续请求会清理超时未访问的 bucket；为 0 时仅按 MaxKeys 淘汰
}

func loadRateLimitConfig() rateLimitConfig {
	cfg := rateLimitConfig{}
	if err := config.ScanFrom(&cfg, "transport.rateLimit"); err != nil {
		logs.Warnw("failed to load rate limit config", "error", err)
	}
	return cfg
}

// rateLimitMiddleware applies process-local limits to this service when enabled.
func rateLimitMiddleware() goendpoint.Middleware {
	cfg := loadRateLimitConfig()
	if !cfg.Enabled || cfg.Every <= 0 || cfg.Burst <= 0 || cfg.MaxKeys <= 0 {
		return nil
	}

	var limiter ratelimit.KeyedLimiter
	limiter, err := ratelimit.NewLocalKeyedRateLimiter(cfg.Every, cfg.Burst, cfg.MaxKeys, cfg.TTL)
	if err != nil {
		logs.Warnw("failed to create local rate limiter", "error", err)
		return nil
	}

	return func(next goendpoint.Endpoint) goendpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			key := rateLimitKey(ctx)
			allowed, err := limiter.Allow(ctx, key)
			if err != nil {
				logs.Ctx(ctx).Warnw("failed to check endpoint rate limit", "key", key, "error", err)
				return nil, errcode.WrapPlatformServiceUnavailable(err)
			}
			if !allowed {
				logs.Ctx(ctx).Debugw("endpoint rate limited", "key", key,
					"rateLimitEvery", cfg.Every.String(), "rateLimitBurst", cfg.Burst)
				return nil, errcode.NewPlatformTooManyRequests()
			}
			return next(ctx, request)
		}
	}
}

// rateLimitKey prefers authenticated user ID and falls back to client IP.
func rateLimitKey(ctx context.Context) string {
	if currentSession, ok := session.SessionFromContext(ctx); ok && currentSession.Subject.UserID != "" {
		return "user:" + currentSession.Subject.UserID
	}
	if ip := contexts.ClientIP(ctx); ip != "" {
		return "ip:" + ip
	}
	return rateLimitAnonymousKey
}
