package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Recovery converts a panic in any downstream handler into a 500 response so a
// single bad request cannot take the server down. The stack trace is logged,
// never returned to the client.
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// The client disconnected mid-write; there is nobody to answer.
				if rec == http.ErrAbortHandler {
					panic(rec)
				}

				logger.ErrorContext(r.Context(), "recovered from panic",
					slog.Any("panic", rec),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("request_id", httpx.RequestIDFromContext(r.Context())),
					slog.String("stack", string(debug.Stack())),
				)

				httpx.Error(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
			}()

			next.ServeHTTP(w, r)
		})
	}
}
