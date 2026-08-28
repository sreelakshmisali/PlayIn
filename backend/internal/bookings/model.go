// Package bookings owns a player's reservation of one turf slot.
//
// It follows the shape already established: service.go holds the rules
// behind a narrow Store interface, repository.go is the only place that
// writes SQL, and handler.go translates HTTP without deciding anything.
//
// Concurrency safety is the one property this package cannot get from Go
// code alone. It comes from PostgreSQL: Repository.CreateBooking reserves a
// slot with the same guarded UPDATE ... WHERE status = 'OPEN' shape every
// other status transition in the owners package already uses, inside one
// transaction with the booking insert, backed by the
// bookings_slot_confirmed_key partial unique index as a second, independent
// guarantee. See migrations/000007_bookings.up.sql for the full reasoning.
package bookings

import (
	"errors"
	"time"
)

// Status is a booking's place in its own small state machine. The set is
// closed and mirrors the bookings_status_chk constraint in migration
// 000007. CONFIRMED is the only status a booking is created with;
// CANCELLED is the only place it can go from there — there is no third
// state in this phase.
type Status string

const (
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)

// Booking is a player's reservation of one turf slot. This is the only
// shape the package ever serialises. It carries no player id: every read in
// this package is already scoped to the caller's own bookings, the same
// reason a player's own profile response needs no separate "is this mine"
// marker.
//
// Date, StartTime and EndTime are joined in from the reserved slot. They are
// not stored on bookings itself — a booking is a reservation of a slot that
// already carries them, not a second copy of when it is.
type Booking struct {
	ID          string     `json:"id"`
	TurfID      string     `json:"turf_id"`
	TurfSlotID  string     `json:"turf_slot_id"`
	Date        string     `json:"date"`
	StartTime   string     `json:"start_time"`
	EndTime     string     `json:"end_time"`
	Status      Status     `json:"status"`
	Price       float64    `json:"price"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
}

// bookingRow is the stored booking, including the player id the repository
// scopes every query to. It never leaves the package.
type bookingRow struct {
	ID          string
	PlayerID    string
	TurfID      string
	TurfSlotID  string
	Date        time.Time
	StartTime   string
	EndTime     string
	Status      Status
	Price       float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CancelledAt *time.Time
}

func (b bookingRow) toBooking() Booking {
	return Booking{
		ID:          b.ID,
		TurfID:      b.TurfID,
		TurfSlotID:  b.TurfSlotID,
		Date:        b.Date.Format(dateLayout),
		StartTime:   b.StartTime,
		EndTime:     b.EndTime,
		Status:      b.Status,
		Price:       b.Price,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		CancelledAt: b.CancelledAt,
	}
}

// dateLayout is the wire format for the joined slot date, matching the
// owners package's own turf_slots convention: ISO 8601 calendar date, no
// time or zone component.
const dateLayout = "2006-01-02"

// Errors returned by the service. Handlers map these to status codes;
// nothing downstream branches on error text.
var (
	// ErrSlotNotBookable means the slot does not exist, or fails the same
	// "available" test turf_slots' own read projection computes: it is not
	// OPEN, or it falls inside a blocked date or blocked time range.
	ErrSlotNotBookable = errors.New("bookings: slot is not open and available")
	// ErrBookingNotFound covers both a booking that does not exist and one
	// that belongs to a different player, answered identically so a guessed
	// id cannot be used to probe for someone else's booking.
	ErrBookingNotFound = errors.New("bookings: booking not found")
	// ErrAlreadyCancelled means the booking's status is already CANCELLED.
	// Cancelling it again is not a valid transition.
	ErrAlreadyCancelled = errors.New("bookings: booking is already cancelled")
)
