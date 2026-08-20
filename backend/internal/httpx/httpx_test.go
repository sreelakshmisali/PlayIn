package httpx

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONWritesBodyAndHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(rec, req, http.StatusCreated, map[string]string{"id": "abc"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["id"] != "abc" {
		t.Errorf("body = %v, want id=abc", body)
	}
}

func TestJSONFailsClosedOnUnencodableValue(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(rec, req, http.StatusOK, math.Inf(1))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestErrorIncludesRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithRequestID(req.Context(), "req-123"))

	Error(rec, req, http.StatusBadRequest, "invalid_request", "Bad input.")

	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body.Error.Code != "invalid_request" {
		t.Errorf("code = %q, want invalid_request", body.Error.Code)
	}
	if body.Error.RequestID != "req-123" {
		t.Errorf("request_id = %q, want req-123", body.Error.RequestID)
	}
}

func TestRequestIDFromContextWhenUnset(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("RequestIDFromContext() = %q, want empty string", got)
	}
}

func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantCode   string
	}{
		{"not found", NotFound, http.StatusNotFound, "not_found"},
		{"method not allowed", MethodNotAllowed, http.StatusMethodNotAllowed, "method_not_allowed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var body ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}
			if body.Error.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tc.wantCode)
			}
		})
	}
}
