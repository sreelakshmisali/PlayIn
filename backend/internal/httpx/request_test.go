package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type payload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func decodeInto(t *testing.T, body, contentType string, dst any) (*httptest.ResponseRecorder, error) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()

	return rec, DecodeJSON(rec, req, dst)
}

func TestDecodeJSON(t *testing.T) {
	var got payload

	rec, err := decodeInto(t, `{"name":"pitch","count":3}`, "application/json", &got)
	if err != nil {
		t.Fatalf("DecodeJSON() returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the recorder untouched", rec.Code)
	}
	if got.Name != "pitch" || got.Count != 3 {
		t.Errorf("decoded = %+v, want {pitch 3}", got)
	}
}

// A charset parameter is legal on Content-Type and must not be rejected.
func TestDecodeJSONAcceptsContentTypeParameters(t *testing.T) {
	var got payload

	if _, err := decodeInto(t, `{"name":"pitch"}`, "application/json; charset=utf-8", &got); err != nil {
		t.Fatalf("DecodeJSON() returned error: %v", err)
	}
}

func TestDecodeJSONRejects(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantMessage string
	}{
		{"wrong content type", `{"name":"a"}`, "text/plain", "Content-Type must be application/json."},
		{"empty body", ``, "application/json", "Body is empty."},
		{"truncated json", `{"name":`, "application/json", "Malformed JSON, the body ended early."},
		{"malformed json", `{"name" "a"}`, "application/json", "Malformed JSON at byte 9."},
		{"wrong field type", `{"count":"three"}`, "application/json", `Field "count" must be a int.`},
		{"unknown field", `{"nope":1}`, "application/json", `Unknown field "nope".`},
		{"trailing object", `{"name":"a"}{"name":"b"}`, "application/json", "Body must contain a single JSON object."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got payload

			rec, err := decodeInto(t, tc.body, tc.contentType, &got)
			if !errors.Is(err, ErrBadRequestBody) {
				t.Fatalf("DecodeJSON() error = %v, want ErrBadRequestBody", err)
			}
			// DecodeJSON answers the request itself, so no handler can decode
			// a body, fail, and forget to write a response.
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}

			var envelope ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decoding the error body failed: %v", err)
			}
			if envelope.Error.Code != "bad_request" {
				t.Errorf("code = %q, want bad_request", envelope.Error.Code)
			}
			if envelope.Error.Message != tc.wantMessage {
				t.Errorf("message = %q, want %q", envelope.Error.Message, tc.wantMessage)
			}
		})
	}
}

// An unbounded body is a memory exhaustion vector, so the reader is capped.
func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	var got payload

	body := `{"name":"` + strings.Repeat("a", maxRequestBody+1) + `"}`
	rec, err := decodeInto(t, body, "application/json", &got)

	if !errors.Is(err, ErrBadRequestBody) {
		t.Fatalf("DecodeJSON() error = %v, want ErrBadRequestBody", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "too large") {
		t.Errorf("body = %s, want it to mention the size limit", rec.Body)
	}
}

func TestValidationError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(WithRequestID(req.Context(), "req-1"))
	rec := httptest.NewRecorder()

	ValidationError(rec, req, []FieldError{
		{Field: "email", Message: "Email is required."},
		{Field: "password", Message: "Password is required."},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	var envelope ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decoding the error body failed: %v", err)
	}
	if envelope.Error.Code != "validation_failed" {
		t.Errorf("code = %q, want validation_failed", envelope.Error.Code)
	}
	if envelope.Error.RequestID != "req-1" {
		t.Errorf("request_id = %q, want req-1", envelope.Error.RequestID)
	}
	if len(envelope.Error.Details) != 2 {
		t.Fatalf("details = %v, want 2 entries", envelope.Error.Details)
	}
	if envelope.Error.Details[0].Field != "email" {
		t.Errorf("details[0].field = %q, want email", envelope.Error.Details[0].Field)
	}
}

// details is omitted on failures that are not validation failures, so clients
// can treat its presence as the signal that per-field reasons exist.
func TestErrorOmitsDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusNotFound, "not_found", "Nope.")

	if strings.Contains(rec.Body.String(), "details") {
		t.Errorf("body = %s, want no details field", rec.Body)
	}
}

func TestSentence(t *testing.T) {
	tests := []struct{ in, want string }{
		{"body is empty", "Body is empty."},
		{"Already capital", "Already capital."},
		{"", "The request body could not be read."},
	}

	for _, tc := range tests {
		if got := sentence(tc.in); got != tc.want {
			t.Errorf("sentence(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
