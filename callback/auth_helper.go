package callback

import (
	"context"

	authpkg "encore.app/auth"
)

type ctxKey struct{}

// withAuthData stores auth data in context (for testing).
func withAuthData(ctx context.Context, data *authpkg.AuthData) context.Context {
	return context.WithValue(ctx, ctxKey{}, data)
}

// getAuthData retrieves auth data from context (test) or Encore's request (production).
func getAuthData(ctx context.Context) *authpkg.AuthData {
	// Check context first (test mode)
	if data, ok := ctx.Value(ctxKey{}).(*authpkg.AuthData); ok && data != nil {
		return data
	}
	// Fall back to Encore's auth
	return authpkg.Data()
}
