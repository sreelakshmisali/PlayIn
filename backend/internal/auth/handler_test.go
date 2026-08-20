package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

const testPrefix = "/api/v1"

// newTestMux mounts the handler exactly as the router does, so these tests
// exercise the real route table including the guards.
func newTestMux(t *testing.T) (*http.ServeMux, *memStore) {
	t.Helper()

	handler, store := newTestHandler(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", httpx.NotFound)
	handler.Routes(mux, testPrefix)

	return mux, store
}

func do(t *testing.T, mux *http.ServeMux, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// decodeSession reads a register, login or refresh response body.
func decodeSession(t *testing.T, rec *httptest.ResponseRecorder) Session {
	t.Helper()

	var session Session
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("decoding the response body failed: %v (body: %s)", err, rec.Body.String())
	}
	return session
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()

	var envelope httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decoding the error body failed: %v (body: %s)", err, rec.Body.String())
	}
	return envelope.Error
}

const registerBody = `{"email":"player@playhub.test","password":"correct horse 7","full_name":"Test Player","role":"PLAYER"}`

func TestHandlerRegister(t *testing.T) {
	mux, _ := newTestMux(t)

	rec := do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
	}

	session := decodeSession(t, rec)
	if session.User.Email != "player@playhub.test" {
		t.Errorf("Email = %q, want player@playhub.test", session.User.Email)
	}
	if session.Tokens.AccessToken == "" {
		t.Error("the response carries no access token")
	}
	// The hash must not appear anywhere in the serialised body.
	if strings.Contains(rec.Body.String(), fakePrefix) {
		t.Error("the response body contains the password hash")
	}
}

func TestHandlerRegisterRejectsDuplicate(t *testing.T) {
	mux, _ := newTestMux(t)

	do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, "")
	rec := do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, "")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if got := decodeError(t, rec).Code; got != "email_taken" {
		t.Errorf("code = %q, want email_taken", got)
	}
}

func TestHandlerRegisterReportsFieldErrors(t *testing.T) {
	mux, _ := newTestMux(t)

	body := `{"email":"nope","password":"short","full_name":"","role":"PLAYER"}`
	rec := do(t, mux, http.MethodPost, testPrefix+"/auth/register", body, "")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	errBody := decodeError(t, rec)
	if errBody.Code != "validation_failed" {
		t.Errorf("code = %q, want validation_failed", errBody.Code)
	}
	if got := fieldNames(errBody.Details); got != "email,full_name,password" {
		t.Errorf("fields = %q, want email,full_name,password", got)
	}
}

func TestHandlerRegisterRejectsAdminSelfAssignment(t *testing.T) {
	mux, _ := newTestMux(t)

	body := `{"email":"a@playhub.test","password":"correct horse 7","full_name":"A B","role":"ADMIN"}`
	rec := do(t, mux, http.MethodPost, testPrefix+"/auth/register", body, "")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandlerRegisterRejectsBadBody(t *testing.T) {
	mux, _ := newTestMux(t)

	tests := []struct {
		name string
		body string
	}{
		{"malformed json", `{"email":`},
		{"unknown field", `{"email":"a@b.test","password":"correct horse 7","full_name":"A B","admin":true}`},
		{"two objects", `{"email":"a@b.test"}{"email":"c@d.test"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, http.MethodPost, testPrefix+"/auth/register", tc.body, "")

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
			}
			if got := decodeError(t, rec).Code; got != "bad_request" {
				t.Errorf("code = %q, want bad_request", got)
			}
		})
	}
}

func TestHandlerLogin(t *testing.T) {
	mux, _ := newTestMux(t)
	do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, "")

	body := `{"email":"Player@PlayHub.test","password":"correct horse 7"}`
	rec := do(t, mux, http.MethodPost, testPrefix+"/auth/login", body, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if decodeSession(t, rec).Tokens.AccessToken == "" {
		t.Error("the response carries no access token")
	}
}

func TestHandlerLoginRejectsBadCredentials(t *testing.T) {
	mux, _ := newTestMux(t)
	do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, "")

	tests := []struct {
		name string
		body string
	}{
		{"wrong password", `{"email":"player@playhub.test","password":"wrong horse 7"}`},
		{"unknown email", `{"email":"nobody@playhub.test","password":"correct horse 7"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, http.MethodPost, testPrefix+"/auth/login", tc.body, "")

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			// Identical code and message for both, so the endpoint cannot be
			// used to find out which addresses are registered.
			errBody := decodeError(t, rec)
			if errBody.Code != "invalid_credentials" {
				t.Errorf("code = %q, want invalid_credentials", errBody.Code)
			}
			if errBody.Message != "Email or password is incorrect." {
				t.Errorf("message = %q, want the generic credentials message", errBody.Message)
			}
		})
	}
}

func TestHandlerMeReturnsTheAuthenticatedUser(t *testing.T) {
	mux, _ := newTestMux(t)
	registered := decodeSession(t, do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, ""))

	rec := do(t, mux, http.MethodGet, testPrefix+"/auth/me", "", registered.Tokens.AccessToken)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var profile Profile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decoding the profile failed: %v", err)
	}
	if profile.ID != registered.User.ID {
		t.Errorf("ID = %q, want %q", profile.ID, registered.User.ID)
	}
	if profile.Email != "player@playhub.test" {
		t.Errorf("Email = %q, want player@playhub.test", profile.Email)
	}
}

func TestHandlerMeRejectsUnauthenticated(t *testing.T) {
	mux, _ := newTestMux(t)
	registered := decodeSession(t, do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, ""))

	tests := []struct {
		name  string
		token string
	}{
		{"no token", ""},
		{"garbage token", "not-a-token"},
		{"refresh token", registered.Tokens.RefreshToken},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, http.MethodGet, testPrefix+"/auth/me", "", tc.token)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", got)
			}
		})
	}
}

func TestHandlerRefreshRotates(t *testing.T) {
	mux, _ := newTestMux(t)
	registered := decodeSession(t, do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, ""))

	body := `{"refresh_token":"` + registered.Tokens.RefreshToken + `"}`
	rec := do(t, mux, http.MethodPost, testPrefix+"/auth/refresh", body, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	rotated := decodeSession(t, rec)
	if rotated.Tokens.RefreshToken == registered.Tokens.RefreshToken {
		t.Error("the refresh token was not rotated")
	}

	// The spent token must be refused the second time round.
	again := do(t, mux, http.MethodPost, testPrefix+"/auth/refresh", body, "")
	if again.Code != http.StatusUnauthorized {
		t.Errorf("replayed refresh status = %d, want %d", again.Code, http.StatusUnauthorized)
	}
}

func TestHandlerLogout(t *testing.T) {
	mux, _ := newTestMux(t)
	registered := decodeSession(t, do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, ""))

	body := `{"refresh_token":"` + registered.Tokens.RefreshToken + `"}`

	rec := do(t, mux, http.MethodPost, testPrefix+"/auth/logout", body, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNoContent, rec.Body)
	}

	// Repeating it stays a 204: ending an already ended session is not a fault.
	if again := do(t, mux, http.MethodPost, testPrefix+"/auth/logout", body, ""); again.Code != http.StatusNoContent {
		t.Errorf("repeated logout status = %d, want %d", again.Code, http.StatusNoContent)
	}

	// The revoked token must no longer buy a new pair.
	refreshed := do(t, mux, http.MethodPost, testPrefix+"/auth/refresh", body, "")
	if refreshed.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout status = %d, want %d", refreshed.Code, http.StatusUnauthorized)
	}
}

func TestHandlerAdminPingEnforcesRole(t *testing.T) {
	mux, store := newTestMux(t)
	registered := decodeSession(t, do(t, mux, http.MethodPost, testPrefix+"/auth/register", registerBody, ""))

	// A PLAYER is authenticated but not authorised.
	rec := do(t, mux, http.MethodGet, testPrefix+"/admin/ping", "", registered.Tokens.AccessToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("player status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "forbidden" {
		t.Errorf("code = %q, want forbidden", got)
	}

	// The same account promoted to ADMIN gets through. The role is read from
	// storage on every request, so the token issued as a PLAYER still works.
	user := store.users[registered.User.ID]
	user.Role = RoleAdmin
	store.users[registered.User.ID] = user

	promoted := do(t, mux, http.MethodGet, testPrefix+"/admin/ping", "", registered.Tokens.AccessToken)
	if promoted.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want %d (body: %s)", promoted.Code, http.StatusOK, promoted.Body)
	}
}

func TestHandlerAdminPingRejectsUnauthenticated(t *testing.T) {
	mux, _ := newTestMux(t)

	rec := do(t, mux, http.MethodGet, testPrefix+"/admin/ping", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandlerRejectsWrongMethod(t *testing.T) {
	mux, _ := newTestMux(t)

	paths := []string{"/auth/register", "/auth/login", "/auth/refresh", "/auth/logout", "/auth/me", "/admin/ping"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := do(t, mux, http.MethodPatch, testPrefix+path, "", "")

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}
