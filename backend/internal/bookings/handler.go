package bookings

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
)

// Handler exposes the booking service over HTTP. Like the other feature
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
// Every path is registered twice, once with its method and once without, for
// the same reason the other feature packages do it: the router's catch-all
// "/" would otherwise turn an unsupported method into a 404.
func (h *Handler) Routes(mux *http.ServeMux, prefix string) {
	// Booking is PLAYER-only. RequireRole names PLAYER exactly, so an OWNER
	// or ADMIN token is refused rather than quietly allowed to book a turf
	// slot for themselves.
	playerOnly := middleware.Chain(auth.RequireAuth(h.authenticator), auth.RequireRole(auth.RolePlayer))

	mux.Handle("POST "+prefix+"/players/me/bookings", playerOnly(http.HandlerFunc(h.Create)))
	mux.Handle("GET "+prefix+"/players/me/bookings", playerOnly(http.HandlerFunc(h.MyBookings)))
	mux.HandleFunc(prefix+"/players/me/bookings", httpx.MethodNotAllowed)

	mux.Handle("GET "+prefix+"/players/me/bookings/{bookingId}", playerOnly(http.HandlerFunc(h.Booking)))
	mux.HandleFunc(prefix+"/players/me/bookings/{bookingId}", httpx.MethodNotAllowed)

	mux.Handle("POST "+prefix+"/players/me/bookings/{bookingId}/cancel", playerOnly(http.HandlerFunc(h.Cancel)))
	mux.HandleFunc(prefix+"/players/me/bookings/{bookingId}/cancel", httpx.MethodNotAllowed)
}

// Create handles POST /api/v1/players/me/bookings.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	var req CreateBookingRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return
	}

	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		httpx.ValidationError(w, r, problems)
		return
	}

	booking, err := h.service.CreateBooking(r.Context(), user.ID, req)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, booking)
}

// MyBookings handles GET /api/v1/players/me/bookings.
func (h *Handler) MyBookings(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	list, err := h.service.MyBookings(r.Context(), user.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, map[string]any{"bookings": list})
}

// Booking handles GET /api/v1/players/me/bookings/{bookingId}.
func (h *Handler) Booking(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	booking, err := h.service.Booking(r.Context(), user.ID, r.PathValue("bookingId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, booking)
}

// Cancel handles POST /api/v1/players/me/bookings/{bookingId}/cancel.
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	user, ok := h.principal(w, r)
	if !ok {
		return
	}

	booking, err := h.service.CancelBooking(r.Context(), user.ID, r.PathValue("bookingId"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, booking)
}

// principal reads the authenticated user the guard put on the context. A
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

// fail maps a service error to a response. Anything unrecognised is a server
// fault: it is logged with its cause and answered with a generic 500, so an
// internal detail never reaches the client.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrSlotNotBookable):
		httpx.Error(w, r, http.StatusConflict, "slot_not_bookable",
			"This slot is no longer open for booking.")
	case errors.Is(err, ErrBookingNotFound):
		httpx.Error(w, r, http.StatusNotFound, "booking_not_found",
			"This booking does not exist.")
	case errors.Is(err, ErrAlreadyCancelled):
		httpx.Error(w, r, http.StatusConflict, "booking_already_cancelled",
			"This booking has already been cancelled.")
	default:
		h.logger.ErrorContext(r.Context(), "booking request failed",
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()),
			slog.String("request_id", httpx.RequestIDFromContext(r.Context())),
		)
		httpx.Error(w, r, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}
