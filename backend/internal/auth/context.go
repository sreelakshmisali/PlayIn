package auth

import "context"

type contextKey string

// principalKey carries the authenticated user through the request context.
const principalKey contextKey = "auth_principal"

// WithPrincipal returns a copy of ctx carrying the authenticated user.
func WithPrincipal(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, principalKey, user)
}

// PrincipalFromContext returns the authenticated user. The boolean is false on
// an unauthenticated request, so a handler mounted without RequireAuth cannot
// mistake a zero User for a real one.
func PrincipalFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(principalKey).(User)
	return user, ok
}
