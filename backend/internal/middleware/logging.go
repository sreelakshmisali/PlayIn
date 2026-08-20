package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// RequestLogger logs one line per request once the handler has returned.
// Server errors log at error level so they surface in alerting.
func RequestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Int("bytes", rw.bytes),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("request_id", httpx.RequestIDFromContext(r.Context())),
			}

			level := slog.LevelInfo
			if rw.status >= http.StatusInternalServerError {
				level = slog.LevelError
			}

			logger.LogAttrs(r.Context(), level, "http request", attrs...)
		})
	}
}
