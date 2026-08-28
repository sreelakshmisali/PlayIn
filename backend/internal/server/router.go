// Package server builds the HTTP router and runs the HTTP server.
package server

import (
	"log/slog"
	"net/http"

	"github.com/orgmelethil/playhub/backend/internal/admin"
	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/bookings"
	"github.com/orgmelethil/playhub/backend/internal/config"
	"github.com/orgmelethil/playhub/backend/internal/health"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
	"github.com/orgmelethil/playhub/backend/internal/owners"
	"github.com/orgmelethil/playhub/backend/internal/players"
)

// APIPrefix is the version prefix every endpoint is mounted under. Breaking
// changes get a new prefix; the old one keeps serving until it is retired.
const APIPrefix = "/api/v1"

// Dependencies are everything the router needs, passed in by the composition
// root in cmd/api. The router constructs nothing itself.
type Dependencies struct {
	Config   *config.Config
	Logger   *slog.Logger
	Health   *health.Handler
	Auth     *auth.Handler
	Players  *players.Handler
	Owners   *owners.Handler
	Admin    *admin.Handler
	Bookings *bookings.Handler
}

// NewRouter builds the application handler: the route table wrapped in the
// middleware stack.
//
// Middleware order matters. Recovery sits outermost so it catches panics from
// everything below it, and RequestID runs before the logger so every log line
// carries a correlation id.
func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	// Anything not matched by a registered pattern gets a JSON 404 rather than
	// net/http's plain-text default.
	mux.HandleFunc("/", httpx.NotFound)

	deps.Health.Routes(mux, APIPrefix)
	deps.Auth.Routes(mux, APIPrefix)
	deps.Players.Routes(mux, APIPrefix)
	deps.Owners.Routes(mux, APIPrefix)
	deps.Admin.Routes(mux, APIPrefix)
	deps.Bookings.Routes(mux, APIPrefix)

	stack := middleware.Chain(
		middleware.Recovery(deps.Logger),
		middleware.RequestID(),
		middleware.RequestLogger(deps.Logger),
		middleware.CORS(deps.Config.HTTP.AllowedOrigins),
	)

	return stack(mux)
}
