// Package middleware holds the cross-cutting HTTP concerns: correlation ids,
// request logging, panic recovery and CORS. Each middleware is a
// func(http.Handler) http.Handler so they compose in any order.
package middleware

import "net/http"

// Middleware wraps an http.Handler with additional behaviour.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to h so that the first argument is the outermost
// wrapper. Chain(a, b)(h) executes a, then b, then h.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// responseWriter records the status code and byte count so the logging
// middleware can report them after the handler returns.
type responseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Unwrap exposes the underlying writer to http.ResponseController.
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
