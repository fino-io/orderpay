package middleware

import "github.com/fino-io/gokit/middleware/accesslog"

func accessLogMiddleware() Middleware {
	return accesslog.HTTPMiddleware()
}
