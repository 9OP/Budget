// Package auth provides JWT validation, PKCE helpers, and context utilities for authentication.
package auth

import "context"

// User holds the authenticated user's identity extracted from the JWT.
type User struct {
	ID    string
	Email string
	Name  string
}

type contextKey int

const contextKeyUser contextKey = iota

// WithUser returns a new context carrying the authenticated user.
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKeyUser, user)
}

// UserFromContext extracts the User injected by the auth middleware.
// Returns (User{}, false) if not present.
func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKeyUser).(User)
	return u, ok && u.ID != ""
}

// UserIDFromContext is a convenience helper that returns just the user UUID.
func UserIDFromContext(ctx context.Context) (string, bool) {
	u, ok := UserFromContext(ctx)
	return u.ID, ok
}
