package auth

import (
	"context"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/token"
)

type claimsKey struct{}

func WithClaims(ctx context.Context, claims token.Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

func ClaimsFrom(ctx context.Context) (token.Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(token.Claims)

	return claims, ok
}
