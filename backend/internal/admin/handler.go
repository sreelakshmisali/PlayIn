// Package admin exposes turf moderation and basic account management to
// ADMIN accounts.
//
// It deliberately owns no model, repository or service of its own: turfs
// belong to the owners package and accounts belong to the auth package
// already, so this package holds only a Handler that calls straight into
// their existing services. The business rules (which status transitions are
// legal, the self-modification guard) live in those services, not here; this
// package's only job is HTTP translation, exactly like every other handler in
// the codebase.
package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
	"github.com/orgmelethil/playhub/backend/internal/owners"
)

// Handler exposes admin moderation and account management over HTTP.
type Handler struct {
	turfs *owners.Service
	users *auth.Service
	// authenticator builds the guard. The package takes it rather than a
	// prebuilt middleware so the guard is declared alongside the routes it
	// protects, matching the players and owners handlers.
	authenticator auth.Authenticator
	logger        *slog.Logger
}

// NewHandler wires a Handler over the existing turf and account services.
func NewHandler(turfs *owners.Service, users *auth.Service, authenticator auth.Authenticator, logger *slog.Logger) *Handler {
	return &Handler{turfs: turfs, users: users, authenticator: authenticator, logger: logger}
}

// Routes registers the package's endpoints under the given prefix.
//
// Every route here requires ADMIN specifically; RequireRole names it exactly,
// so a PLAYER or OWNER token is refused rather than granted admin behaviour by
// accident. This is a separate guard chain from the OWNER-only one in the
// owners package: ADMIN is not added as a bypass to the existing owner routes,
// it gets its own namespace instead.
func (h *Handler) Routes(mux *http.ServeMux, prefix string) {
	adminOnly := middleware.Chain(auth.RequireAuth(h.authenticator), auth.RequireRole(auth.RoleAdmin))

	mux.Handle("GET "+prefix+"/admin/turfs/pending", adminOnly(http.HandlerFunc(h.PendingTurfs)))

	mux.Handle("GET "+prefix+"/admin/turfs/{turfId}", adminOnly(http.HandlerFunc(h.Turf)))
	// One method-less pattern covers both /admin/turfs/pending and
	// /admin/turfs/{turfId}. Registering a second one for the literal path
	// would conflict: against "GET /admin/turfs/{turfId}" neither is more
	// specific, since one is narrower by method and the other by path. This
	// mirrors the same conflict, and the same fix, in players.Handler.Routes.
	mux.HandleFunc(prefix+"/admin/turfs/{turfId}", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/admin/turfs/{turfId}/approve", adminOnly(http.HandlerFunc(h.ApproveTurf)))
	mux.HandleFunc(prefix+"/admin/turfs/{turfId}/approve", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/admin/turfs/{turfId}/reject", adminOnly(http.HandlerFunc(h.RejectTurf)))
	mux.HandleFunc(prefix+"/admin/turfs/{turfId}/reject", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/admin/turfs/{turfId}/suspend", adminOnly(http.HandlerFunc(h.SuspendTurf)))
	mux.HandleFunc(prefix+"/admin/turfs/{turfId}/suspend", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/admin/turfs/{turfId}/restore", adminOnly(http.HandlerFunc(h.RestoreTurf)))
	mux.HandleFunc(prefix+"/admin/turfs/{turfId}/restore", httpx.MethodNotAllowed)

	mux.Handle("GET "+prefix+"/admin/users", adminOnly(http.HandlerFunc(h.Users)))
	mux.HandleFunc(prefix+"/admin/users", httpx.MethodNotAllowed)

	mux.Handle("GET "+prefix+"/admin/users/{userId}", adminOnly(http.HandlerFunc(h.User)))
	mux.HandleFunc(prefix+"/admin/users/{userId}", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/admin/users/{userId}/deactivate", adminOnly(http.HandlerFunc(h.DeactivateUser)))
	mux.HandleFunc(prefix+"/admin/users/{userId}/deactivate", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/admin/users/{userId}/reactivate", adminOnly(http.HandlerFunc(h.ReactivateUser)))
	mux.HandleFunc(prefix+"/admin/users/{userId}/reactivate", httpx.MethodNotAllowed)
}

// PendingTurfs handles GET /api/v1/admin/turfs/pending.
func (h *Handler) PendingTurfs(w http.ResponseWriter, r *http.Request) {
	turfs, err := h.turfs.PendingTurfs(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"turfs": turfs})
}

// Turf handles GET /api/v1/admin/turfs/{turfId}. Unlike the owner-facing and
// public turf reads, this is not scoped by owner or status: an admin can look
// up any turf by id.
func (h *Handler) Turf(w http.ResponseWriter, r *http.Request) {
	turf, err := h.turfs.AdminTurf(r.Context(), r.PathValue("turfId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// ApproveTurf handles POST /api/v1/admin/turfs/{turfId}/approve.
func (h *Handler) ApproveTurf(w http.ResponseWriter, r *http.Request) {
	admin, ok := h.principal(w, r)
	if !ok {
		return
	}

	turf, err := h.turfs.ApproveTurf(r.Context(), r.PathValue("turfId"), admin.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// RejectTurf handles POST /api/v1/admin/turfs/{turfId}/reject.
func (h *Handler) RejectTurf(w http.ResponseWriter, r *http.Request) {
	admin, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req owners.ModerateTurfRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}
	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.turfs.RejectTurf(r.Context(), r.PathValue("turfId"), admin.ID, req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// SuspendTurf handles POST /api/v1/admin/turfs/{turfId}/suspend.
func (h *Handler) SuspendTurf(w http.ResponseWriter, r *http.Request) {
	admin, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req owners.ModerateTurfRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}
	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	turf, err := h.turfs.SuspendTurf(r.Context(), r.PathValue("turfId"), admin.ID, req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// RestoreTurf handles POST /api/v1/admin/turfs/{turfId}/restore.
func (h *Handler) RestoreTurf(w http.ResponseWriter, r *http.Request) {
	admin, ok := h.principal(w, r)
	if !ok {
		return
	}

	turf, err := h.turfs.RestoreTurf(r.Context(), r.PathValue("turfId"), admin.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, turf)
}

// Users handles GET /api/v1/admin/users?limit=&offset=.
func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit")
	offset := queryInt(r, "offset")

	page, err := h.users.ListUsers(r.Context(), limit, offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, page)
}

// User handles GET /api/v1/admin/users/{userId}.
func (h *Handler) User(w http.ResponseWriter, r *http.Request) {
	user, err := h.users.AdminUser(r.Context(), r.PathValue("userId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, user)
}

// DeactivateUser handles POST /api/v1/admin/users/{userId}/deactivate.
func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, false)
}

// ReactivateUser handles POST /api/v1/admin/users/{userId}/reactivate.
func (h *Handler) ReactivateUser(w http.ResponseWriter, r *http.Request) {
	h.setActive(w, r, true)
}

func (h *Handler) setActive(w http.ResponseWriter, r *http.Request, active bool) {
	admin, ok := h.principal(w, r)
	if !ok {
		return
	}

	user, err := h.users.SetUserActive(r.Context(), admin.ID, r.PathValue("userId"), active)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, user)
}

// principal reads the authenticated admin the guard put on the context. A
// miss is a wiring mistake, so it answers 401 rather than proceeding with a
// zero id.
func (h *Handler) principal(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, http.StatusUnauthorized, "unauthorized", "A bearer access token is required.")
		return auth.User{}, false
	}
	return user, true
}

// queryInt reads a non-negative integer query parameter. A missing or
// unparsable value is treated as 0 and left for the service to apply its own
// default; this handler does not decide what a sensible page size is.
func queryInt(r *http.Request, name string) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// fail maps a service error to a response. Anything unrecognised is a server
// fault: it is logged with its cause and answered with a generic 500, so an
// internal detail never reaches the client.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, owners.ErrTurfNotFound):
		httpx.Error(w, r, http.StatusNotFound, "turf_not_found", "This turf does not exist.")
	case errors.Is(err, owners.ErrInvalidStatusTransition):
		httpx.Error(w, r, http.StatusConflict, "invalid_status_transition",
			"This action is not valid for the turf's current status.")
	case errors.Is(err, auth.ErrUserNotFound):
		httpx.Error(w, r, http.StatusNotFound, "user_not_found", "This user does not exist.")
	case errors.Is(err, auth.ErrCannotModifySelf):
		httpx.Error(w, r, http.StatusConflict, "cannot_modify_self",
			"You cannot perform this action on your own account.")
	default:
		h.logger.ErrorContext(r.Context(), "admin request failed",
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
			slog.String("request_id", httpx.RequestIDFromContext(r.Context())),
		)
		httpx.Error(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
