package middleware

import "net/http"

type RouteMiddleware func(Route) []Middleware

// Route describes a single HTTP binding at registration time.
type Route struct {
	MethodName  string
	BindingName string
	Method      string
	Path        string
}

// WrapRoute applies route-specific middleware without affecting the global chain.
func WrapRoute(route Route, next http.Handler) http.Handler {
	return Chain(next, RouteMiddlewares(route)...)
}

func RouteMiddlewares(route Route) []Middleware {
	return route.Use(bindingMiddleware)
}

func (route Route) Use(groups ...RouteMiddleware) []Middleware {
	middlewares := make([]Middleware, 0, len(groups))
	for _, group := range groups {
		if group != nil {
			middlewares = append(middlewares, group(route)...)
		}
	}

	return middlewares
}

func bindingMiddleware(route Route) []Middleware {
	switch route.BindingName {
	case "CreateOrder":
		return createOrderMiddleware(route)
	case "GetOrder":
		return getOrderMiddleware(route)
	case "ListOrders":
		return listOrdersMiddleware(route)
	case "CancelOrder":
		return cancelOrderMiddleware(route)
	case "CreatePayment":
		return createPaymentMiddleware(route)
	case "ListAuditEvents":
		return listAuditEventsMiddleware(route)
	default:
		return nil
	}
}

func createOrderMiddleware(Route) []Middleware {
	return nil
}

func getOrderMiddleware(Route) []Middleware {
	return nil
}

func listOrdersMiddleware(Route) []Middleware {
	return nil
}

func cancelOrderMiddleware(Route) []Middleware {
	return nil
}

func createPaymentMiddleware(Route) []Middleware {
	return nil
}

func listAuditEventsMiddleware(Route) []Middleware {
	return nil
}
