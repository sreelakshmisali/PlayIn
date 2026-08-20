package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// RequestIDHeader carries the correlation id in and out of the service.
const RequestIDHeader = "X-Request-ID"

// RequestID attaches a correlation id to every request. An inbound
// X-Request-ID is reused so a trace survives across services; otherwise a new
// one is generated. The id is echoed back on the response.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if id == "" || len(id) > 128 {
				id = newRequestID()
			}

			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(httpx.WithRequestID(r.Context(), id)))
		})
	}
}

func newRequestID() string {
	buf := make([]byte, 16)
	// crypto/rand.Read never returns an error as of Go 1.24.
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
