package httpx

import "context"

type contextKey string

// requestIDKey carries the per-request correlation id through the context.
const requestIDKey contextKey = "request_id"

// WithRequestID returns a copy of ctx carrying the given request id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request id, or an empty string if unset.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
