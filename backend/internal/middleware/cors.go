package middleware

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// CORS answers browser preflight requests and adds the response headers the
// frontend needs when it is served from a different origin than the API.
// Origins come from configuration; there is no wildcard fallback.
func CORS(allowedOrigins []string) Middleware {
	allowMethods := strings.Join([]string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions,
	}, ", ")
	allowHeaders := strings.Join([]string{"Content-Type", "Authorization", RequestIDHeader}, ", ")
	maxAge := strconv.Itoa(int((12 * time.Hour).Seconds()))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && slices.Contains(allowedOrigins, origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Expose-Headers", RequestIDHeader)
				h.Add("Vary", "Origin")

				if r.Method == http.MethodOptions {
					h.Set("Access-Control-Allow-Methods", allowMethods)
					h.Set("Access-Control-Allow-Headers", allowHeaders)
					h.Set("Access-Control-Max-Age", maxAge)
					h.Add("Vary", "Access-Control-Request-Method")
					h.Add("Vary", "Access-Control-Request-Headers")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
