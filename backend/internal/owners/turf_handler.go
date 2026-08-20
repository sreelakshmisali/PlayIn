package owners

import (
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// CreateTurf handles POST /api/v1/owners/me/turfs.
func (h *Handler) CreateTurf(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SaveTurfRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.service.CreateTurf(r.Context(), user.ID, req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, turf)
}

// MyTurfs handles GET /api/v1/owners/me/turfs.
func (h *Handler) MyTurfs(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	turfs, err := h.service.MyTurfs(r.Context(), user.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"turfs": turfs})
}

// MyTurf handles GET /api/v1/owners/me/turfs/{turfId}.
func (h *Handler) MyTurf(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	turf, err := h.service.MyTurf(r.Context(), user.ID, r.PathValue("turfId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// UpdateTurf handles PUT /api/v1/owners/me/turfs/{turfId}.
func (h *Handler) UpdateTurf(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SaveTurfRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.service.UpdateTurf(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// DeleteTurf handles DELETE /api/v1/owners/me/turfs/{turfId}.
func (h *Handler) DeleteTurf(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteTurf(r.Context(), user.ID, r.PathValue("turfId")); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SubmitTurf handles POST /api/v1/owners/me/turfs/{turfId}/submit.
func (h *Handler) SubmitTurf(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	turf, err := h.service.SubmitTurf(r.Context(), user.ID, r.PathValue("turfId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// AddTurfSport handles POST /api/v1/owners/me/turfs/{turfId}/sports.
//
// It attaches a sport, or is a no-op if the turf already has it. There is no
// per-turf attribute to change on a repeat call the way a player's position
// can change, so unlike the player sports endpoint this is a pure add rather
// than an upsert with meaningful new data.
func (h *Handler) AddTurfSport(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SetTurfSportRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.service.SetTurfSport(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// RemoveTurfSport handles DELETE /api/v1/owners/me/turfs/{turfId}/sports/{sportId}.
func (h *Handler) RemoveTurfSport(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	turf, err := h.service.RemoveTurfSport(r.Context(), user.ID, r.PathValue("turfId"), r.PathValue("sportId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// AddTurfAmenity handles POST /api/v1/owners/me/turfs/{turfId}/amenities.
func (h *Handler) AddTurfAmenity(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SetTurfAmenityRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.service.SetTurfAmenity(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// RemoveTurfAmenity handles DELETE /api/v1/owners/me/turfs/{turfId}/amenities/{amenityId}.
func (h *Handler) RemoveTurfAmenity(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	turf, err := h.service.RemoveTurfAmenity(r.Context(), user.ID, r.PathValue("turfId"), r.PathValue("amenityId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// AddTurfImage handles POST /api/v1/owners/me/turfs/{turfId}/images.
func (h *Handler) AddTurfImage(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req AddTurfImageRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.service.AddTurfImage(r.Context(), user.ID, r.PathValue("turfId"), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// RemoveTurfImage handles DELETE /api/v1/owners/me/turfs/{turfId}/images/{imageId}.
func (h *Handler) RemoveTurfImage(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	turf, err := h.service.RemoveTurfImage(r.Context(), user.ID, r.PathValue("turfId"), r.PathValue("imageId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// PublicTurfs handles GET /api/v1/turfs. No token: browsing is public and only
// ever surfaces APPROVED turfs.
func (h *Handler) PublicTurfs(w http.ResponseWriter, r *http.Request) {
	turfs, err := h.service.PublicTurfs(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"turfs": turfs})
}

// PublicTurf handles GET /api/v1/turfs/{turfId}.
func (h *Handler) PublicTurf(w http.ResponseWriter, r *http.Request) {
	turf, err := h.service.PublicTurf(r.Context(), r.PathValue("turfId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}
