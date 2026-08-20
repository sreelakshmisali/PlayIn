package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
)

// Handler exposes the auth service over HTTP. Like the health handler it holds
// no logic: it decodes, validates, calls the service and maps errors to status
// codes.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler wires a Handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// Routes registers the package's endpoints under the given prefix.
//
// Each path is registered twice, once with its method and once without, for the
// same reason the health package does it: the router's catch-all "/" would
// otherwise turn an unsupported method into a 404.
//
// The protected routes are wrapped here rather than centrally, so a route and
// the guard it runs behind are declared in the same place.
func (h *Handler) Routes(mux *http.ServeMux, prefix string) {
	guard := RequireAuth(h.service)

	mux.HandleFunc("POST "+prefix+"/auth/register", h.Register)
	mux.HandleFunc(prefix+"/auth/register", httpx.MethodNotAllowed)

	mux.HandleFunc("POST "+prefix+"/auth/login", h.Login)
	mux.HandleFunc(prefix+"/auth/login", httpx.MethodNotAllowed)

	mux.HandleFunc("POST "+prefix+"/auth/refresh", h.Refresh)
	mux.HandleFunc(prefix+"/auth/refresh", httpx.MethodNotAllowed)

	mux.HandleFunc("POST "+prefix+"/auth/logout", h.Logout)
	mux.HandleFunc(prefix+"/auth/logout", httpx.MethodNotAllowed)

	mux.Handle("GET "+prefix+"/auth/me", guard(http.HandlerFunc(h.Me)))
	mux.HandleFunc(prefix+"/auth/me", httpx.MethodNotAllowed)

	// The one mount that exercises RequireRole until the Phase 2 admin surface
	// exists. It reports the caller's identity and nothing else.
	adminOnly := middleware.Chain(guard, RequireRole(RoleAdmin))
	mux.Handle("GET "+prefix+"/admin/ping", adminOnly(http.HandlerFunc(h.AdminPing)))
	mux.HandleFunc(prefix+"/admin/ping", httpx.MethodNotAllowed)
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	session, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.JSON(w, r, http.StatusCreated, session)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	session, err := h.service.Login(r.Context(), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.JSON(w, r, http.StatusOK, session)
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	session, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.JSON(w, r, http.StatusOK, session)
}

// Logout handles POST /api/v1/auth/logout. It revokes the refresh token and
// answers 204 whether or not the token was still live.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Me handles GET /api/v1/auth/me. RequireAuth has already resolved the user,
// so this only projects it.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := PrincipalFromContext(r.Context())
	if !ok {
		unauthorised(w, r, "A bearer access token is required.")
		return
	}

	httpx.JSON(w, r, http.StatusOK, user.Profile())
}

// AdminPing handles GET /api/v1/admin/ping. It exists to prove the role guard
// in front of it works and carries no product behaviour.
func (h *Handler) AdminPing(w http.ResponseWriter, r *http.Request) {
	user, ok := PrincipalFromContext(r.Context())
	if !ok {
		unauthorised(w, r, "A bearer access token is required.")
		return
	}

	httpx.JSON(w, r, http.StatusOK, map[string]string{
		"status": "ok",
		"user":   user.ID,
		"role":   string(user.Role),
	})
}

// fail maps a service error to a response. Anything unrecognised is a server
// fault: it is logged with its cause and answered with a generic 500, so an
// internal detail never reaches the client.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(w, r, http.StatusConflict, "email_taken",
			"An account with this email already exists.")
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUserNotFound):
		// Both answer the same way, so a caller cannot tell a wrong password
		// from an address that was never registered.
		httpx.Error(w, r, http.StatusUnauthorized, "invalid_credentials",
			"Email or password is incorrect.")
	case errors.Is(err, ErrInvalidToken):
		httpx.Error(w, r, http.StatusUnauthorized, "invalid_token",
			"The token is invalid, expired or has already been used.")
	case errors.Is(err, ErrAccountInactive):
		httpx.Error(w, r, http.StatusForbidden, "account_inactive",
			"This account has been deactivated.")
	default:
		h.logger.ErrorContext(r.Context(), "auth request failed",
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
			slog.String("request_id", httpx.RequestIDFromContext(r.Context())),
		)
		httpx.Error(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
