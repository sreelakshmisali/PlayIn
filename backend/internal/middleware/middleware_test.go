package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("brewing"))
	})
}

func TestChainAppliesOutermostFirst(t *testing.T) {
	var order []string

	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	handler := Chain(mark("first"), mark("second"))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if got := strings.Join(order, ","); got != "first,second,handler" {
		t.Errorf("execution order = %q, want first,second,handler", got)
	}
}

func TestRequestIDGeneratesWhenAbsent(t *testing.T) {
	var seen string
	handler := RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = httpx.RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if len(seen) != 32 {
		t.Errorf("generated request id = %q, want 32 hex characters", seen)
	}
	if rec.Header().Get(RequestIDHeader) != seen {
		t.Errorf("response header = %q, want %q", rec.Header().Get(RequestIDHeader), seen)
	}
}

func TestRequestIDReusesInboundHeader(t *testing.T) {
	var seen string
	handler := RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = httpx.RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "upstream-id")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "upstream-id" {
		t.Errorf("request id = %q, want upstream-id", seen)
	}
}

func TestRequestIDRejectsOversizedHeader(t *testing.T) {
	var seen string
	handler := RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = httpx.RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, strings.Repeat("x", 200))

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if len(seen) != 32 {
		t.Errorf("request id = %q, want a freshly generated id", seen)
	}
}

func TestRequestLoggerRecordsStatusAndBytes(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	RequestLogger(logger)(okHandler()).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/teapot", nil))

	out := buf.String()
	for _, want := range []string{"status=418", "bytes=7", "path=/teapot"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output = %q, want it to contain %q", out, want)
		}
	}
}

func TestRecoveryConvertsPanicTo500(t *testing.T) {
	handler := Recovery(discardLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("response body leaks the panic value")
	}
}

func TestRecoveryPassesThroughWhenNoPanic(t *testing.T) {
	rec := httptest.NewRecorder()
	Recovery(discardLogger())(okHandler()).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	CORS([]string{"http://localhost:5173"})(okHandler()).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the request origin", got)
	}
}

func TestCORSIgnoresUnknownOrigin(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example")

	CORS([]string{"http://localhost:5173"})(okHandler()).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it unset", got)
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want the handler to still run", rec.Code)
	}
}

func TestCORSAnswersPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	CORS([]string{"http://localhost:5173"})(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), http.MethodPost) {
		t.Error("preflight response is missing the allowed methods")
	}
}
