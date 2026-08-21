package owners

import (
	"errors"
	"time"
)

// SlotStatus is a slot's own management state, set directly by the owner. It
// is one input to a slot's derived availability; the other is whether the
// slot falls inside a blocked date or a blocked time range (see Slot).
//
// Phase 6 adds a BOOKED value here and nothing else about this package
// changes: a booking reserves a slot with `UPDATE turf_slots SET status =
// 'BOOKED' WHERE id = $1 AND status = 'OPEN'`, the same guarded-write shape
// every turf status transition in this package already uses.
type SlotStatus string

const (
	SlotStatusOpen    SlotStatus = "OPEN"
	SlotStatusBlocked SlotStatus = "BLOCKED"
)

// Slot is one bookable window on one turf on one date.
//
// Available is derived, never stored: it is true exactly when Status is OPEN
// and the slot falls inside neither a blocked date nor a blocked time range.
// It is computed fresh by the repository on every read, so a block added
// after the slot was generated takes effect immediately without anything
// having to go back and update this row.
type Slot struct {
	ID        string     `json:"id"`
	Date      string     `json:"date"`
	StartTime string     `json:"start_time"`
	EndTime   string     `json:"end_time"`
	Price     float64    `json:"price"`
	Status    SlotStatus `json:"status"`
	Available bool       `json:"available"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// slotRow is the stored slot, including the surrogate turf key. It never
// leaves the package.
type slotRow struct {
	ID        string
	TurfID    string
	Date      time.Time
	StartTime string
	EndTime   string
	Price     float64
	Status    SlotStatus
	Available bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s slotRow) toSlot() Slot {
	return Slot{
		ID:        s.ID,
		Date:      s.Date.Format(dateLayout),
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		Price:     s.Price,
		Status:    s.Status,
		Available: s.Available,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// dateLayout is the wire format for every plain date this package accepts or
// returns: ISO 8601 calendar date, no time or zone component.
const dateLayout = "2006-01-02"

// BlockedDate is a whole day an owner has taken off availability, independent
// of whether slots have been generated for it.
type BlockedDate struct {
	ID     string `json:"id"`
	Date   string `json:"date"`
	Reason string `json:"reason,omitempty"`
}

type blockedDateRow struct {
	ID     string
	Date   time.Time
	Reason string
}

func (b blockedDateRow) toBlockedDate() BlockedDate {
	return BlockedDate{ID: b.ID, Date: b.Date.Format(dateLayout), Reason: b.Reason}
}

// BlockedTimeRange is part of one day taken off availability, e.g. a
// maintenance window.
type BlockedTimeRange struct {
	ID        string `json:"id"`
	Date      string `json:"date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Reason    string `json:"reason,omitempty"`
}

type blockedTimeRangeRow struct {
	ID        string
	Date      time.Time
	StartTime string
	EndTime   string
	Reason    string
}

func (b blockedTimeRangeRow) toBlockedTimeRange() BlockedTimeRange {
	return BlockedTimeRange{
		ID: b.ID, Date: b.Date.Format(dateLayout),
		StartTime: b.StartTime, EndTime: b.EndTime, Reason: b.Reason,
	}
}

// slotSettingsFields are the writable slot-configuration columns on turfs:
// everything an owner sets before slots can be generated.
type slotSettingsFields struct {
	DurationMinutes int32
	Price           float64
}

// Errors returned by the slot and availability side of the service. Handlers
// map these to status codes; nothing downstream branches on error text.
var (
	// ErrSlotNotFound covers both a slot that does not exist and one that
	// belongs to a different owner's turf, answered identically for the same
	// reason ErrTurfNotFound is.
	ErrSlotNotFound = errors.New("owners: slot not found")
	// ErrSlotSettingsNotConfigured means generation was attempted before the
	// owner set both a slot duration and a price.
	ErrSlotSettingsNotConfigured = errors.New("owners: slot duration and price must be configured before generating slots")
	// ErrInvalidDateRange means a from/to pair was backwards or spans more
	// dates than a single request may generate or list.
	ErrInvalidDateRange = errors.New("owners: invalid date range")
	// ErrBlockedDateNotFound means no blocked-date row with this id exists
	// for this owner's turf.
	ErrBlockedDateNotFound = errors.New("owners: blocked date not found")
	// ErrDateAlreadyBlocked means this turf already has this exact date
	// blocked.
	ErrDateAlreadyBlocked = errors.New("owners: date is already blocked")
	// ErrBlockedTimeRangeNotFound means no blocked-time-range row with this
	// id exists for this owner's turf.
	ErrBlockedTimeRangeNotFound = errors.New("owners: blocked time range not found")
	// ErrTimeRangeOverlapsBlock means the requested block overlaps a time
	// range already blocked on the same turf and date.
	ErrTimeRangeOverlapsBlock = errors.New("owners: time range overlaps an existing block")
)
