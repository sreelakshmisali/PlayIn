package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestServiceCheckHealthy(t *testing.T) {
	svc := NewService(stubPinger{}, "1.2.3", "development")

	report := svc.Check(context.Background())

	if !report.Healthy() {
		t.Errorf("Healthy() = false, want true")
	}
	if report.Status != StatusOK {
		t.Errorf("Status = %q, want %q", report.Status, StatusOK)
	}
	if report.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", report.Version)
	}
	if report.Checks["database"].Status != StatusOK {
		t.Errorf("database check = %q, want %q", report.Checks["database"].Status, StatusOK)
	}
	if report.Timestamp.IsZero() {
		t.Error("Timestamp is zero, want the probe time")
	}
}

func TestServiceCheckDegradedWhenDatabaseFails(t *testing.T) {
	svc := NewService(stubPinger{err: errors.New("connection refused")}, "dev", "development")

	report := svc.Check(context.Background())

	if report.Healthy() {
		t.Error("Healthy() = true, want false")
	}
	if got := report.Checks["database"].Error; got != "connection refused" {
		t.Errorf("database error = %q, want %q", got, "connection refused")
	}
}

func TestServiceCheckWithoutDatabase(t *testing.T) {
	svc := NewService(nil, "dev", "development")

	report := svc.Check(context.Background())

	if report.Status != StatusDegraded {
		t.Errorf("Status = %q, want %q", report.Status, StatusDegraded)
	}
}

func TestHandlerGet(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   Status
	}{
		{"healthy", nil, http.StatusOK, StatusOK},
		{"degraded", errors.New("down"), http.StatusServiceUnavailable, StatusDegraded},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(NewService(stubPinger{err: tc.pingErr}, "dev", "test"))

			mux := http.NewServeMux()
			handler.Routes(mux, "/api/v1")

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want JSON", ct)
			}

			var body Report
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}
			if body.Status != tc.wantBody {
				t.Errorf("body status = %q, want %q", body.Status, tc.wantBody)
			}
		})
	}
}
