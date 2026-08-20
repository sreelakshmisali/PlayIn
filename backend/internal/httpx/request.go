package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"
)

// maxRequestBody caps how much of a request body is read. Every endpoint in
// this API takes a small JSON object, so a megabyte is generous and stops an
// unbounded body from consuming memory.
const maxRequestBody = 1 << 20

// ErrBadRequestBody is returned by DecodeJSON when the body cannot be used.
// Handlers translate it into a 400; the wrapped text explains what was wrong.
var ErrBadRequestBody = errors.New("bad request body")

// DecodeJSON reads exactly one JSON object from the request into dst.
//
// It is stricter than a bare json.Decoder on purpose: unknown fields are
// rejected so a typo in a client payload fails loudly instead of being
// silently ignored, and trailing content is rejected so two concatenated
// objects cannot be smuggled through as one.
//
// On failure it writes the 400 itself and returns the error, so a handler's
// decode step is one line and no handler can forget to answer. The returned
// error wraps ErrBadRequestBody.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	err := decodeJSON(w, r, dst)
	if err != nil {
		BadRequest(w, r, sentence(describeDecodeError(err)))
	}
	return err
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		if media, _, _ := strings.Cut(ct, ";"); strings.TrimSpace(media) != "application/json" {
			return fmt.Errorf("%w: Content-Type must be application/json", ErrBadRequestBody)
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: %s", ErrBadRequestBody, describeDecodeError(err))
	}
	if dec.More() {
		return fmt.Errorf("%w: body must contain a single JSON object", ErrBadRequestBody)
	}
	return nil
}

// BadRequest writes a 400 for a malformed request body.
func BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusBadRequest, "bad_request", message)
}

// sentence turns a lower-case fragment into a message fit for a client.
func sentence(fragment string) string {
	if fragment == "" {
		return "The request body could not be read."
	}
	runes := []rune(fragment)
	return string(unicode.ToUpper(runes[0])) + string(runes[1:]) + "."
}

func describeDecodeError(err error) string {
	// decodeJSON wraps its own refusals with ErrBadRequestBody and a message
	// that is already client-safe, so those pass through unchanged.
	if errors.Is(err, ErrBadRequestBody) {
		_, msg, _ := strings.Cut(err.Error(), ": ")
		return msg
	}

	var syntax *json.SyntaxError
	var unmarshalType *json.UnmarshalTypeError
	var maxBytes *http.MaxBytesError

	switch {
	case errors.As(err, &syntax):
		return fmt.Sprintf("malformed JSON at byte %d", syntax.Offset)
	case errors.As(err, &unmarshalType):
		return fmt.Sprintf("field %q must be a %s", unmarshalType.Field, unmarshalType.Type)
	case errors.As(err, &maxBytes):
		return "body is too large"
	case errors.Is(err, io.ErrUnexpectedEOF):
		// A truncated body: the decoder ran out of input mid-value.
		return "malformed JSON, the body ended early"
	case errors.Is(err, io.EOF):
		return "body is empty"
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return "unknown field " + strings.TrimPrefix(err.Error(), "json: unknown field ")
	default:
		return "body could not be parsed"
	}
}
