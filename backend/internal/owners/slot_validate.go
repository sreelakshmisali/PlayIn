package owners

import (
	"strings"
	"time"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Input limits for the slot and availability endpoints.
const (
	minSlotDuration = 15
	maxSlotDuration = 240
	maxSlotPrice    = 1_000_000
	// maxGenerateDays bounds a single generate call: enough for a month at a
	// time, not enough to let one request materialise years of rows.
	maxGenerateDays = 31
	// maxListDays bounds the owner's list/availability range query.
	maxListDays    = 92
	maxBlockReason = 250
)

// SlotSettingsRequest is the body of PATCH
// /owners/me/turfs/{turfId}/slot-settings. Both fields are required: slots
// cannot be generated with only one configured.
type SlotSettingsRequest struct {
	SlotDurationMinutes int32   `json:"slot_duration_minutes"`
	SlotPrice           float64 `json:"slot_price"`
}

func (r SlotSettingsRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	switch {
	case r.SlotDurationMinutes < minSlotDuration || r.SlotDurationMinutes > maxSlotDuration:
		errs = append(errs, field("slot_duration_minutes", "Slot duration must be between 15 and 240 minutes."))
	case r.SlotDurationMinutes%15 != 0:
		errs = append(errs, field("slot_duration_minutes", "Slot duration must be a multiple of 15 minutes."))
	}

	if r.SlotPrice < 0 || r.SlotPrice > maxSlotPrice {
		errs = append(errs, field("slot_price", "Price must be zero or a positive amount."))
	}

	return errs
}

func (r SlotSettingsRequest) fields() slotSettingsFields {
	return slotSettingsFields{DurationMinutes: r.SlotDurationMinutes, Price: r.SlotPrice}
}

// GenerateSlotsRequest is the body of POST
// /owners/me/turfs/{turfId}/slots/generate.
type GenerateSlotsRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (r *GenerateSlotsRequest) Normalise() {
	r.From = strings.TrimSpace(r.From)
	r.To = strings.TrimSpace(r.To)
}

func (r GenerateSlotsRequest) Validate() []httpx.FieldError {
	return validateDateRange(r.From, r.To)
}

// SlotRangeQuery is the from/to pair the owner's slot list and availability
// views take as query parameters.
type SlotRangeQuery struct {
	From string
	To   string
}

func (r SlotRangeQuery) Validate() []httpx.FieldError {
	return validateDateRange(r.From, r.To)
}

func validateDateRange(from, to string) []httpx.FieldError {
	var errs []httpx.FieldError

	fromDate, fromErr := parseDate(from)
	if fromErr != nil {
		errs = append(errs, field("from", "From date must be a valid date (YYYY-MM-DD)."))
	}
	toDate, toErr := parseDate(to)
	if toErr != nil {
		errs = append(errs, field("to", "To date must be a valid date (YYYY-MM-DD)."))
	}
	if fromErr != nil || toErr != nil {
		return errs
	}

	if toDate.Before(fromDate) {
		errs = append(errs, field("to", "To date must not be before the from date."))
		return errs
	}
	if toDate.Sub(fromDate) > (maxGenerateDaysDuration) {
		errs = append(errs, field("to", "The date range must not span more than 31 days."))
	}
	return errs
}

const maxGenerateDaysDuration = (maxGenerateDays - 1) * 24 * time.Hour

func parseDate(value string) (time.Time, error) {
	return time.Parse(dateLayout, value)
}

// SetSlotStatusRequest is the body of PATCH
// /owners/me/turfs/{turfId}/slots/{slotId}.
type SetSlotStatusRequest struct {
	Status SlotStatus `json:"status"`
}

func (r *SetSlotStatusRequest) Normalise() {
	r.Status = SlotStatus(strings.ToUpper(strings.TrimSpace(string(r.Status))))
}

func (r SetSlotStatusRequest) Validate() []httpx.FieldError {
	if r.Status != SlotStatusOpen && r.Status != SlotStatusBlocked {
		return []httpx.FieldError{field("status", "Status must be OPEN or BLOCKED.")}
	}
	return nil
}

// BlockDateRequest is the body of POST /owners/me/turfs/{turfId}/blocked-dates.
type BlockDateRequest struct {
	Date   string `json:"date"`
	Reason string `json:"reason"`
}

func (r *BlockDateRequest) Normalise() {
	r.Date = strings.TrimSpace(r.Date)
	r.Reason = strings.TrimSpace(r.Reason)
}

func (r BlockDateRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError
	if _, err := parseDate(r.Date); err != nil {
		errs = append(errs, field("date", "Date must be a valid date (YYYY-MM-DD)."))
	}
	if n := len([]rune(r.Reason)); n > maxBlockReason {
		errs = append(errs, field("reason", "Reason must be 250 characters or fewer."))
	}
	return errs
}

// BlockTimeRangeRequest is the body of POST
// /owners/me/turfs/{turfId}/blocked-time-ranges.
type BlockTimeRangeRequest struct {
	Date      string `json:"date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Reason    string `json:"reason"`
}

func (r *BlockTimeRangeRequest) Normalise() {
	r.Date = strings.TrimSpace(r.Date)
	r.StartTime = strings.TrimSpace(r.StartTime)
	r.EndTime = strings.TrimSpace(r.EndTime)
	r.Reason = strings.TrimSpace(r.Reason)
}

func (r BlockTimeRangeRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	if _, err := parseDate(r.Date); err != nil {
		errs = append(errs, field("date", "Date must be a valid date (YYYY-MM-DD)."))
	}
	if !timePattern.MatchString(r.StartTime) {
		errs = append(errs, field("start_time", "Start time must be a 24-hour HH:MM value."))
	}
	if !timePattern.MatchString(r.EndTime) {
		errs = append(errs, field("end_time", "End time must be a 24-hour HH:MM value."))
	}
	if len(errs) == 0 && r.EndTime <= r.StartTime {
		errs = append(errs, field("end_time", "End time must be after the start time."))
	}
	if n := len([]rune(r.Reason)); n > maxBlockReason {
		errs = append(errs, field("reason", "Reason must be 250 characters or fewer."))
	}
	return errs
}
