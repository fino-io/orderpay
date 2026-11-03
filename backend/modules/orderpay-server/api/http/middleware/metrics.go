package middleware

import (
	"net/http"

	gokitmetrics "github.com/fino-io/gokit/metrics"
	"github.com/gorilla/mux"
)

func MetricsMiddleware(instrument gokitmetrics.HTTPMiddleware) mux.MiddlewareFunc {
	return mux.MiddlewareFunc(instrument(func(request *http.Request) string {
		route := mux.CurrentRoute(request)
		if route == nil {
			return ""
		}

		template, _ := route.GetPathTemplate()
		return template
	}))
}
