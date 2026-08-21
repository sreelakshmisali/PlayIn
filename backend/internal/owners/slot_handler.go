package owners

import (
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// UpdateSlotSettings handles PATCH /api/v1/owners/me/turfs/{turfId}/slot-settings.
func (h *Handler) UpdateSlotSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SlotSettingsRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.service.UpdateSlotSettings(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// GenerateSlots handles POST /api/v1/owners/me/turfs/{turfId}/slots/generate.
func (h *Handler) GenerateSlots(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req GenerateSlotsRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	slots, err := h.service.GenerateSlots(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"slots": slots})
}

// MySlots handles GET /api/v1/owners/me/turfs/{turfId}/slots?from=&to=.
func (h *Handler) MySlots(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	query := SlotRangeQuery{From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to")}
	if problems := query.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	slots, err := h.service.SlotsInRange(r.Context(), user.ID, r.PathValue("turfId"), query)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"slots": slots})
}

// SetSlotStatus handles PATCH /api/v1/owners/me/turfs/{turfId}/slots/{slotId}.
func (h *Handler) SetSlotStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SetSlotStatusRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	slot, err := h.service.SetSlotStatus(r.Context(), user.ID, r.PathValue("turfId"), r.PathValue("slotId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, slot)
}

// DeleteSlot handles DELETE /api/v1/owners/me/turfs/{turfId}/slots/{slotId}.
func (h *Handler) DeleteSlot(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteSlot(r.Context(), user.ID, r.PathValue("turfId"), r.PathValue("slotId")); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BlockedDates handles GET /api/v1/owners/me/turfs/{turfId}/blocked-dates.
func (h *Handler) BlockedDates(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	dates, err := h.service.BlockedDates(r.Context(), user.ID, r.PathValue("turfId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"blocked_dates": dates})
}

// BlockDate handles POST /api/v1/owners/me/turfs/{turfId}/blocked-dates.
func (h *Handler) BlockDate(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req BlockDateRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	blocked, err := h.service.BlockDate(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, blocked)
}

// UnblockDate handles DELETE
// /api/v1/owners/me/turfs/{turfId}/blocked-dates/{blockedDateId}.
func (h *Handler) UnblockDate(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	err := h.service.UnblockDate(r.Context(), user.ID, r.PathValue("turfId"), r.PathValue("blockedDateId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BlockedTimeRanges handles GET
// /api/v1/owners/me/turfs/{turfId}/blocked-time-ranges.
func (h *Handler) BlockedTimeRanges(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	ranges, err := h.service.BlockedTimeRanges(r.Context(), user.ID, r.PathValue("turfId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"blocked_time_ranges": ranges})
}

// BlockTimeRange handles POST
// /api/v1/owners/me/turfs/{turfId}/blocked-time-ranges.
func (h *Handler) BlockTimeRange(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req BlockTimeRangeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	blocked, err := h.service.BlockTimeRange(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, blocked)
}

// UnblockTimeRange handles DELETE
// /api/v1/owners/me/turfs/{turfId}/blocked-time-ranges/{blockedTimeRangeId}.
func (h *Handler) UnblockTimeRange(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	err := h.service.UnblockTimeRange(r.Context(), user.ID, r.PathValue("turfId"), r.PathValue("blockedTimeRangeId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PublicAvailability handles GET /api/v1/turfs/{turfId}/availability?date=.
// No token: browsing availability is public, matching PublicTurf, and only
// ever surfaces an APPROVED turf's slots.
func (h *Handler) PublicAvailability(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if _, err := parseDate(date); err != nil {
		httpx.ValidationError(w, r, []httpx.FieldError{field("date", "Date must be a valid date (YYYY-MM-DD).")})
		return
	}

	slots, err := h.service.PublicSlotsForDate(r.Context(), r.PathValue("turfId"), date)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"date": date, "slots": slots})
}
