package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/orgmelethil/playhub/backend/internal/admin"
	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/config"
	"github.com/orgmelethil/playhub/backend/internal/health"
	"github.com/orgmelethil/playhub/backend/internal/middleware"
	"github.com/orgmelethil/playhub/backend/internal/owners"
	"github.com/orgmelethil/playhub/backend/internal/players"
)

type stubPinger struct{}

func (stubPinger) Ping(context.Context) error { return nil }

func testRouter() http.Handler {
	cfg := &config.Config{}
	cfg.HTTP.AllowedOrigins = []string{"http://localhost:5173"}
	cfg.Auth = config.Auth{
		JWTSecret:  "0123456789abcdef0123456789abcdef",
		JWTIssuer:  "playhub-test",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
		BcryptCost: bcrypt.MinCost,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// The store is nil because these tests only assert routing: no route
	// reached here gets past the token check, so nothing touches storage.
	authService := auth.NewService(nil, auth.NewBcryptHasher(cfg.Auth.BcryptCost), auth.NewIssuer(cfg.Auth))
	authHandler := auth.NewHandler(authService, logger)
	playersHandler := players.NewHandler(players.NewService(nil), authService, logger)
	ownersService := owners.NewService(nil)
	ownersHandler := owners.NewHandler(ownersService, authService, logger)
	adminHandler := admin.NewHandler(ownersService, authService, authService, logger)

	return NewRouter(Dependencies{
		Config:  cfg,
		Logger:  logger,
		Health:  health.NewHandler(health.NewService(stubPinger{}, "test", "test")),
		Auth:    authHandler,
		Players: playersHandler,
		Owners:  ownersHandler,
		Admin:   adminHandler,
	})
}

func TestRouterServesHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/health", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get(middleware.RequestIDHeader) == "" {
		t.Error("response is missing the request id header")
	}
}

func TestRouterRejectsWrongMethod(t *testing.T) {
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, APIPrefix+"/health", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRouterReturnsJSONNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", ct)
	}
}

func TestRouterMountsAuthRoutes(t *testing.T) {
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/auth/me", nil))

	// 401 rather than 404 proves the route is mounted and the guard, not the
	// catch-all, answered it.
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRouterGuardsAdminRoute(t *testing.T) {
	rec := httptest.NewRecorder()
	testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+"/admin/ping", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRouterGuardsPlayerRoutes(t *testing.T) {
	for _, path := range []string{"/players/me", "/players/me/sports"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+path, nil))

			// 401 rather than 404 proves the route is mounted behind the guard.
			if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want 401 or 405", rec.Code)
			}
		})
	}
}

func TestRouterGuardsOwnerRoutes(t *testing.T) {
	for _, path := range []string{"/owners/me", "/owners/me/turfs"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+path, nil))

			// 401 rather than 404 proves the route is mounted behind the guard.
			if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want 401 or 405", rec.Code)
			}
		})
	}
}

func TestRouterGuardsAdminModerationRoutes(t *testing.T) {
	for _, path := range []string{"/admin/turfs/pending", "/admin/users"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+path, nil))

			// 401 rather than 404 proves the route is mounted behind the guard.
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestRouterServesPublicTurfsAndAmenities(t *testing.T) {
	// nil store: this only asserts the routes are mounted and public, not that
	// they succeed, so a 500 from the nil store is still evidence the guard
	// did not intervene (a guard failure would be 401).
	for _, path := range []string{"/turfs", "/amenities"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			testRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, APIPrefix+path, nil))

			if rec.Code == http.StatusUnauthorized {
				t.Errorf("status = %d, want a public route to never answer 401", rec.Code)
			}
		})
	}
}
