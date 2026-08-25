package bookings

import (
	"strings"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// CreateBookingRequest is the body of POST /players/me/bookings.
//
// It carries only the slot: not the turf. The slot id alone already
// addresses one exact row, and the reservation query derives its turf (and
// checks that turf is APPROVED) from the slot itself, so there is no client-
// supplied turf id that could ever disagree with it.
type CreateBookingRequest struct {
	TurfSlotID string `json:"turf_slot_id"`
}

// Normalise trims the request. It runs before validation so trailing space
// is not reported as an error.
func (r *CreateBookingRequest) Normalise() {
	r.TurfSlotID = strings.TrimSpace(r.TurfSlotID)
}

func (r CreateBookingRequest) Validate() []httpx.FieldError {
	if r.TurfSlotID == "" {
		return []httpx.FieldError{field("turf_slot_id", "Turf slot id is required.")}
	}
	return nil
}

func field(name, message string) httpx.FieldError {
	return httpx.FieldError{Field: name, Message: message}
}
