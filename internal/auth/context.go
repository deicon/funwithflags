package auth

import "context"

type contextKey string

const userContextKey contextKey = "funwithflags_auth_user"

// WithUser stores the authenticated user on the context.
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext extracts the authenticated user from context if present.
func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey).(User)
	return user, ok
}
