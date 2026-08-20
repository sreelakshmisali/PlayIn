// Package httpx holds the HTTP helpers shared by every handler: a single JSON
// writer and a single error envelope. Handlers use these so responses stay
// consistent across the API.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// FieldError names one rejected input field. Clients render these next to the
// offending control instead of parsing the top-level message.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorBody is the payload returned for every failed request.
// Details is omitted unless the failure is a validation failure.
type ErrorBody struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	RequestID string       `json:"request_id,omitempty"`
	Details   []FieldError `json:"details,omitempty"`
}

// ErrorResponse is the envelope wrapping an ErrorBody.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// JSON writes v as a JSON response with the given status code.
// It encodes into a buffer first so an encoding failure cannot emit a
// half-written body under an already-sent 200.
func JSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		slog.ErrorContext(r.Context(), "encode response body", slog.String("error", err.Error()))
		http.Error(w, `{"error":{"code":"internal_error","message":"Something went wrong."}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response body", slog.String("error", err.Error()))
	}
}

// Error writes a JSON error envelope, attaching the request id when present.
func Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	JSON(w, r, status, ErrorResponse{Error: ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: RequestIDFromContext(r.Context()),
	}})
}

// ValidationError writes a 422 carrying the per-field reasons a request was
// rejected. Validation is the one failure mode where a single message is not
// enough for the caller to fix the request.
func ValidationError(w http.ResponseWriter, r *http.Request, fields []FieldError) {
	JSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{Error: ErrorBody{
		Code:      "validation_failed",
		Message:   "The request contains invalid fields.",
		RequestID: RequestIDFromContext(r.Context()),
		Details:   fields,
	}})
}

// NotFound is the router's fallback for unknown paths.
func NotFound(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusNotFound, "not_found", "The requested resource does not exist.")
}

// MethodNotAllowed is the router's fallback for unsupported methods.
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "This method is not supported for the requested resource.")
}
