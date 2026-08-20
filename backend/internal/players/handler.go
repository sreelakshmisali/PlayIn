package players

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
)

// Handler exposes the player service over HTTP. Like the auth and health
// handlers it holds no logic: it decodes, validates, calls the service and maps
// errors to status codes.
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
// Each path is registered twice, once with its method and once without, for the
// same reason the other feature packages do it: the router's catch-all "/"
// would otherwise turn an unsupported method into a 404.
func (h *Handler) Routes(mux *http.ServeMux, prefix string) {
	// Managing a player profile is PLAYER-only. RequireRole names PLAYER
	// exactly, so an OWNER or ADMIN token is refused rather than quietly
	// granted a profile it has no use for.
	playerOnly := middleware.Chain(auth.RequireAuth(h.authenticator), auth.RequireRole(auth.RolePlayer))

	mux.Handle("GET "+prefix+"/players/me", playerOnly(http.HandlerFunc(h.Me)))
	mux.Handle("PUT "+prefix+"/players/me", playerOnly(http.HandlerFunc(h.Save)))
	mux.Handle("PATCH "+prefix+"/players/me", playerOnly(http.HandlerFunc(h.Patch)))

	mux.Handle("POST "+prefix+"/players/me/sports", playerOnly(http.HandlerFunc(h.AddSport)))
	mux.HandleFunc(prefix+"/players/me/sports", httpx.MethodNotAllowed)

	mux.Handle("DELETE "+prefix+"/players/me/sports/{sportId}", playerOnly(http.HandlerFunc(h.RemoveSport)))
	mux.HandleFunc(prefix+"/players/me/sports/{sportId}", httpx.MethodNotAllowed)

	// Public. A profile is meant to be found by other players, and it carries
	// nothing that needs a token to read.
	//
	// The literal /players/me patterns above win over this wildcard because
	// ServeMux prefers the more specific pattern. "me" is not a valid UUID, so
	// no real user id can be shadowed by them.
	mux.HandleFunc("GET "+prefix+"/players/{userId}", h.Public)

	// One method-less pattern covers both /players/me and /players/{userId}.
	// Registering a second one for the literal path would conflict: against
	// "GET /players/{userId}" neither is more specific, since one is narrower
	// by method and the other by path.
	mux.HandleFunc(prefix+"/players/{userId}", httpx.MethodNotAllowed)

	mux.HandleFunc("GET "+prefix+"/sports", h.Sports)
	mux.HandleFunc(prefix+"/sports", httpx.MethodNotAllowed)
}

// Sports handles GET /api/v1/sports.
func (h *Handler) Sports(w http.ResponseWriter, r *http.Request) {
	sports, err := h.service.Sports(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.JSON(w, r, http.StatusOK, map[string]any{"sports": sports})
}

// Me handles GET /api/v1/players/me.
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

// Public handles GET /api/v1/players/{userId}.
//
// It is the same projection the owner receives. A player's profile is public by
// design, and the projection carries no account data, so there is nothing to
// withhold from a stranger.
func (h *Handler) Public(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.Profile(r.Context(), r.PathValue("userId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.JSON(w, r, http.StatusOK, profile)
}

// Save handles PUT /api/v1/players/me. It creates the profile or replaces it.
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

// Patch handles PATCH /api/v1/players/me.
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

// AddSport handles POST /api/v1/players/me/sports.
//
// It adds a preferred sport, or changes the position on one already chosen.
// Making it an upsert rather than refusing a duplicate keeps the client to one
// verb for "this is how I play this sport", which is the only thing it ever
// wants to say.
func (h *Handler) AddSport(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req SetSportRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	profile, err := h.service.SetSport(r.Context(), user.ID, req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.JSON(w, r, http.StatusOK, profile)
}

// RemoveSport handles DELETE /api/v1/players/me/sports/{sportId}.
func (h *Handler) RemoveSport(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	profile, err := h.service.RemoveSport(r.Context(), user.ID, r.PathValue("sportId"))
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
	var position *PositionError

	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Error(w, r, http.StatusNotFound, "profile_not_found",
			"This player has no profile yet.")
	case errors.As(err, &position):
		// A rejected position is a field problem, and the client cannot fix it
		// without knowing what is allowed, so the message lists the choices.
		httpx.ValidationError(w, r, []httpx.FieldError{
			{Field: "position", Message: position.Message()},
		})
	case errors.Is(err, ErrSportNotFound):
		httpx.Error(w, r, http.StatusNotFound, "sport_not_found",
			"That sport does not exist or is no longer available.")
	case errors.Is(err, ErrSportNotPreferred):
		httpx.Error(w, r, http.StatusNotFound, "sport_not_preferred",
			"That sport is not one of your preferred sports.")
	default:
		h.logger.ErrorContext(r.Context(), "player request failed",
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
			slog.String("request_id", httpx.RequestIDFromContext(r.Context())),
		)
		httpx.Error(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
