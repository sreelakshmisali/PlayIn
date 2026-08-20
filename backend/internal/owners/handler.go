package owners

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
)

// Handler exposes the owner service over HTTP. Like the auth and players
// handlers it holds no logic: it decodes, validates, calls the service and
// maps errors to status codes.
type Handler struct {
	service *Service
	// authenticator builds the guards. The package takes it rather than a
	// prebuilt middleware so each route below declares the guard it runs
	// behind, in the same place as the route itself.
	authenticator auth.Authenticator
	logger        *slog.Logger
}

// NewHandler wires a Handler.
func NewHandler(service *Service, authenticator auth.Authenticator, logger *slog.Logger) *Handler {
	return &Handler{service: service, authenticator: authenticator, logger: logger}
}

// Routes registers the package's endpoints under the given prefix.
//
// Each path is registered twice, once with its method and once without, for
// the same reason the other feature packages do it: the router's catch-all "/"
// would otherwise turn an unsupported method into a 404.
//
// Every owner-facing turf path lives under /owners/me/turfs, entirely separate
// from the public /turfs namespace. Unlike a player's own profile and the
// public player lookup, which share one path parameterised by user id, there
// is no wildcard collision to route around here.
func (h *Handler) Routes(mux *http.ServeMux, prefix string) {
	ownerOnly := middleware.Chain(auth.RequireAuth(h.authenticator), auth.RequireRole(auth.RoleOwner))

	mux.Handle("GET "+prefix+"/owners/me", ownerOnly(http.HandlerFunc(h.Me)))
	mux.Handle("PUT "+prefix+"/owners/me", ownerOnly(http.HandlerFunc(h.Save)))
	mux.Handle("PATCH "+prefix+"/owners/me", ownerOnly(http.HandlerFunc(h.Patch)))
	mux.HandleFunc(prefix+"/owners/me", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/owners/me/turfs", ownerOnly(http.HandlerFunc(h.CreateTurf)))
	mux.Handle("GET "+prefix+"/owners/me/turfs", ownerOnly(http.HandlerFunc(h.MyTurfs)))
	mux.HandleFunc(prefix+"/owners/me/turfs", httpx.MethodNotAllowed)

	mux.Handle("GET "+prefix+"/owners/me/turfs/{turfId}", ownerOnly(http.HandlerFunc(h.MyTurf)))
	mux.Handle("PUT "+prefix+"/owners/me/turfs/{turfId}", ownerOnly(http.HandlerFunc(h.UpdateTurf)))
	mux.Handle("DELETE "+prefix+"/owners/me/turfs/{turfId}", ownerOnly(http.HandlerFunc(h.DeleteTurf)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/owners/me/turfs/{turfId}/submit", ownerOnly(http.HandlerFunc(h.SubmitTurf)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}/submit", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/owners/me/turfs/{turfId}/sports", ownerOnly(http.HandlerFunc(h.AddTurfSport)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}/sports", httpx.MethodNotAllowed)
	mux.Handle("DELETE "+prefix+"/owners/me/turfs/{turfId}/sports/{sportId}", ownerOnly(http.HandlerFunc(h.RemoveTurfSport)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}/sports/{sportId}", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/owners/me/turfs/{turfId}/amenities", ownerOnly(http.HandlerFunc(h.AddTurfAmenity)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}/amenities", httpx.MethodNotAllowed)
	mux.Handle("DELETE "+prefix+"/owners/me/turfs/{turfId}/amenities/{amenityId}", ownerOnly(http.HandlerFunc(h.RemoveTurfAmenity)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}/amenities/{amenityId}", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/owners/me/turfs/{turfId}/images", ownerOnly(http.HandlerFunc(h.AddTurfImage)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}/images", httpx.MethodNotAllowed)
	mux.Handle("DELETE "+prefix+"/owners/me/turfs/{turfId}/images/{imageId}", ownerOnly(http.HandlerFunc(h.RemoveTurfImage)))
	mux.HandleFunc(prefix+"/owners/me/turfs/{turfId}/images/{imageId}", httpx.MethodNotAllowed)

	// Public. Browsing turfs and the amenities they can offer needs no token.
	mux.HandleFunc("GET "+prefix+"/amenities", h.Amenities)
	mux.HandleFunc(prefix+"/amenities", httpx.MethodNotAllowed)

	mux.HandleFunc("GET "+prefix+"/turfs", h.PublicTurfs)
	mux.HandleFunc(prefix+"/turfs", httpx.MethodNotAllowed)

	mux.HandleFunc("GET "+prefix+"/turfs/{turfId}", h.PublicTurf)
	mux.HandleFunc(prefix+"/turfs/{turfId}", httpx.MethodNotAllowed)
}

// Amenities handles GET /api/v1/amenities.
func (h *Handler) Amenities(w http.ResponseWriter, r *http.Request) {
	amenities, err := h.service.Amenities(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"amenities": amenities})
}

// Me handles GET /api/v1/owners/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	profile, err := h.service.Profile(r.Context(), user.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, profile)
}

// Save handles PUT /api/v1/owners/me. It creates the profile or replaces it.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SaveProfileRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	profile, created, err := h.service.SaveProfile(r.Context(), user.ID, req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, r, status, profile)
}

// Patch handles PATCH /api/v1/owners/me.
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req PatchProfileRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if req.Empty() {
		httpx.BadRequest(w, r, "The request body must contain at least one field to change.")
		return
	}
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	profile, err := h.service.PatchProfile(r.Context(), user.ID, req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, profile)
}

// principal reads the authenticated user the guard put on the context. A miss
// is a wiring mistake, so it answers 401 rather than proceeding with a zero id.
func (h *Handler) principal(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, http.StatusUnauthorized, "unauthorized", "A bearer access token is required.")
		return auth.User{}, false
	}
	return user, true
}

// fail maps a service error to a response. Anything unrecognised is a server
// fault: it is logged with its cause and answered with a generic 500, so an
// internal detail never reaches the client.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrOwnerProfileNotFound):
		httpx.Error(w, r, http.StatusNotFound, "owner_profile_not_found",
			"Set up your owner profile before listing a turf.")
	case errors.Is(err, ErrTurfNotFound):
		httpx.Error(w, r, http.StatusNotFound, "turf_not_found",
			"This turf does not exist or is not visible to you.")
	case errors.Is(err, ErrTurfNameTaken):
		httpx.Error(w, r, http.StatusConflict, "turf_name_taken",
			"You already have a turf with this name.")
	case errors.Is(err, ErrInvalidStatusTransition):
		httpx.Error(w, r, http.StatusConflict, "invalid_status_transition",
			"This turf cannot be submitted for review from its current status.")
	case errors.Is(err, ErrSportNotFound):
		httpx.Error(w, r, http.StatusNotFound, "sport_not_found",
			"That sport does not exist or is no longer available.")
	case errors.Is(err, ErrTurfSportNotFound):
		httpx.Error(w, r, http.StatusNotFound, "turf_sport_not_found",
			"This turf does not have that sport.")
	case errors.Is(err, ErrAmenityNotFound):
		httpx.Error(w, r, http.StatusNotFound, "amenity_not_found",
			"That amenity does not exist or is no longer available.")
	case errors.Is(err, ErrTurfAmenityNotFound):
		httpx.Error(w, r, http.StatusNotFound, "turf_amenity_not_found",
			"This turf does not have that amenity.")
	case errors.Is(err, ErrTurfImageNotFound):
		httpx.Error(w, r, http.StatusNotFound, "turf_image_not_found",
			"This turf does not have that image.")
	case errors.Is(err, ErrTooManyImages):
		httpx.Error(w, r, http.StatusUnprocessableEntity, "too_many_images",
			"This turf already has the maximum number of images.")
	default:
		h.logger.ErrorContext(r.Context(), "owner request failed",
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
			slog.String("request_id", httpx.RequestIDFromContext(r.Context())),
		)
		httpx.Error(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
