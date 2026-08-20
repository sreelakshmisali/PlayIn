package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubAuthenticator answers with a fixed user or a fixed error, so the
// middleware can be tested without a service behind it.
type stubAuthenticator struct {
	user User
	err  error
}

func (s stubAuthenticator) Authenticate(context.Context, string) (User, error) {
	return s.user, s.err
}

// okHandler records that the request reached the far side of the middleware.
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAuthPassesTheUserThrough(t *testing.T) {
	want := User{ID: "user-1", Role: RoleOwner, IsActive: true}

	var got User
	handler := RequireAuth(stubAuthenticator{user: want})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				t.Error("the request context carries no principal")
			}
			got = principal
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got.ID != want.ID {
		t.Errorf("principal ID = %q, want %q", got.ID, want.ID)
	}
}

func TestRequireAuthRejectsBadHeaders(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"absent", ""},
		{"no scheme", "some-token"},
		{"wrong scheme", "Basic dXNlcjpwYXNz"},
		{"empty credential", "Bearer "},
		{"whitespace credential", "Bearer    "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			handler := RequireAuth(stubAuthenticator{user: User{ID: "user-1"}})(okHandler(&reached))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if reached {
				t.Error("the request reached the handler behind the guard")
			}
		})
	}
}

// The scheme is case insensitive per RFC 7235, so a client sending "bearer"
// must not be turned away.
func TestRequireAuthAcceptsLowercaseScheme(t *testing.T) {
	reached := false
	handler := RequireAuth(stubAuthenticator{user: User{ID: "user-1"}})(okHandler(&reached))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer some-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !reached {
		t.Errorf("status = %d, want the request to reach the handler", rec.Code)
	}
}

func TestRequireAuthMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid token", ErrInvalidToken, http.StatusUnauthorized},
		{"inactive account", ErrAccountInactive, http.StatusForbidden},
		{"unexpected failure", errors.New("database is on fire"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			handler := RequireAuth(stubAuthenticator{err: tc.err})(okHandler(&reached))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer some-token")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if reached {
				t.Error("the request reached the handler behind the guard")
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		held       Role
		allowed    []Role
		wantStatus int
	}{
		{"exact match", RoleAdmin, []Role{RoleAdmin}, http.StatusOK},
		{"one of several", RoleOwner, []Role{RoleOwner, RoleAdmin}, http.StatusOK},
		{"player denied admin", RolePlayer, []Role{RoleAdmin}, http.StatusForbidden},
		{"owner denied admin", RoleOwner, []Role{RoleAdmin}, http.StatusForbidden},
		{"admin is not implicitly allowed", RoleAdmin, []Role{RolePlayer}, http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			handler := RequireRole(tc.allowed...)(okHandler(&reached))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(WithPrincipal(req.Context(), User{ID: "user-1", Role: tc.held}))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if reached != (tc.wantStatus == http.StatusOK) {
				t.Errorf("handler reached = %t, want %t", reached, tc.wantStatus == http.StatusOK)
			}
		})
	}
}

// RequireRole mounted without RequireAuth in front of it is a wiring mistake.
// Refusing is the safe reading; letting the request through would make the
// route public.
func TestRequireRoleRefusesWithoutPrincipal(t *testing.T) {
	reached := false
	handler := RequireRole(RoleAdmin)(okHandler(&reached))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if reached {
		t.Error("the request reached the handler behind the guard")
	}
}

func TestPrincipalFromContextWithoutPrincipal(t *testing.T) {
	if _, ok := PrincipalFromContext(context.Background()); ok {
		t.Error("PrincipalFromContext() reported a principal on a bare context")
	}
}
