package model

import (
	"context"
	"errors"
)

var (
	ErrConflict    = errors.New("business conflict")
	ErrRateLimited = errors.New("merchant write rate limited")
	ErrUnavailable = errors.New("payment channel disabled")
)

type Principal struct {
	RequestID  string
	MerchantID string
	ActorID    string
	Admin      bool
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFrom(ctx context.Context) Principal {
	principal, _ := ctx.Value(principalKey{}).(Principal)
	return principal
}
