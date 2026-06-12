// Package auth provides JWT validation, PKCE helpers, and context utilities for authentication.
package auth

import "context"

type contextKey int

const contextKeyUserID contextKey = iota

// WithUserID returns a new context carrying the authenticated user's UUID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

// UserIDFromContext extracts the user UUID injected by the auth middleware.
// Returns ("", false) if not present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKeyUserID).(string)
	return id, ok && id != ""
}
