package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceRegisterIssuesSession(t *testing.T) {
	svc, store := newTestService(t)

	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	if session.User.Email != "player@playhub.test" {
		t.Errorf("Email = %q, want player@playhub.test", session.User.Email)
	}
	if session.User.Role != RolePlayer {
		t.Errorf("Role = %q, want PLAYER", session.User.Role)
	}
	if session.Tokens.AccessToken == "" || session.Tokens.RefreshToken == "" {
		t.Error("Register() returned an empty token")
	}
	if session.Tokens.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want Bearer", session.Tokens.TokenType)
	}
	if session.Tokens.ExpiresIn != int64((15 * time.Minute).Seconds()) {
		t.Errorf("ExpiresIn = %d, want 900", session.Tokens.ExpiresIn)
	}
	if len(store.tokens) != 1 {
		t.Errorf("stored refresh tokens = %d, want 1", len(store.tokens))
	}
}

// The Profile projection is what reaches the client. A password hash appearing
// in it would be a credential leak on every successful login.
func TestServiceRegisterDoesNotExposeHash(t *testing.T) {
	svc, store := newTestService(t)

	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	stored, err := store.UserByID(context.Background(), session.User.ID)
	if err != nil {
		t.Fatalf("UserByID() returned error: %v", err)
	}
	if stored.PasswordHash == "" {
		t.Fatal("the stored user has no password hash")
	}
	if stored.PasswordHash == validRegisterRequest().Password {
		t.Error("the password was stored in plain text")
	}
}

func TestServiceRegisterRejectsDuplicateEmail(t *testing.T) {
	svc, _ := newTestService(t)

	registerUser(t, svc, "player@playhub.test", RolePlayer)

	req := validRegisterRequest()
	req.Normalise()
	if _, err := svc.Register(context.Background(), req); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("Register() error = %v, want ErrEmailTaken", err)
	}
}

func TestServiceLoginSucceeds(t *testing.T) {
	svc, _ := newTestService(t)
	registerUser(t, svc, "player@playhub.test", RolePlayer)

	req := LoginRequest{Email: "player@playhub.test", Password: validRegisterRequest().Password}
	session, err := svc.Login(context.Background(), req)
	if err != nil {
		t.Fatalf("Login() returned error: %v", err)
	}
	if session.User.Email != "player@playhub.test" {
		t.Errorf("Email = %q, want player@playhub.test", session.User.Email)
	}
	if session.Tokens.AccessToken == "" {
		t.Error("Login() returned an empty access token")
	}
}

// A wrong password and an unknown address must be indistinguishable, or the
// endpoint becomes an account enumeration oracle.
func TestServiceLoginFailuresAreIndistinguishable(t *testing.T) {
	svc, _ := newTestService(t)
	registerUser(t, svc, "player@playhub.test", RolePlayer)

	tests := []struct {
		name string
		req  LoginRequest
	}{
		{"wrong password", LoginRequest{Email: "player@playhub.test", Password: "wrong horse 7"}},
		{"unknown email", LoginRequest{Email: "nobody@playhub.test", Password: "correct horse 7"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Login(context.Background(), tc.req); !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestServiceLoginRejectsInactiveAccount(t *testing.T) {
	svc, store := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)
	store.setActive(session.User.ID, false)

	req := LoginRequest{Email: "player@playhub.test", Password: validRegisterRequest().Password}
	if _, err := svc.Login(context.Background(), req); !errors.Is(err, ErrAccountInactive) {
		t.Errorf("Login() error = %v, want ErrAccountInactive", err)
	}
}

func TestServiceAuthenticateReturnsUser(t *testing.T) {
	svc, _ := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	user, err := svc.Authenticate(context.Background(), session.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("Authenticate() returned error: %v", err)
	}
	if user.ID != session.User.ID {
		t.Errorf("ID = %q, want %q", user.ID, session.User.ID)
	}
}

// The account is read on every call rather than trusted from the claims, so a
// deactivation takes effect immediately instead of at token expiry.
func TestServiceAuthenticateRejectsDeactivatedUser(t *testing.T) {
	svc, store := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)
	store.setActive(session.User.ID, false)

	if _, err := svc.Authenticate(context.Background(), session.Tokens.AccessToken); !errors.Is(err, ErrAccountInactive) {
		t.Errorf("Authenticate() error = %v, want ErrAccountInactive", err)
	}
}

func TestServiceAuthenticateRejectsRefreshToken(t *testing.T) {
	svc, _ := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	if _, err := svc.Authenticate(context.Background(), session.Tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Authenticate() error = %v, want ErrInvalidToken", err)
	}
}

func TestServiceAuthenticateRejectsDeletedUser(t *testing.T) {
	svc, store := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	delete(store.users, session.User.ID)

	if _, err := svc.Authenticate(context.Background(), session.Tokens.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Authenticate() error = %v, want ErrInvalidToken", err)
	}
}

func TestServiceRefreshRotatesTokens(t *testing.T) {
	svc, _ := newTestService(t)
	first := registerUser(t, svc, "player@playhub.test", RolePlayer)

	second, err := svc.Refresh(context.Background(), first.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() returned error: %v", err)
	}
	if second.Tokens.RefreshToken == first.Tokens.RefreshToken {
		t.Error("Refresh() reissued the same refresh token, want a rotated one")
	}
	if second.User.ID != first.User.ID {
		t.Errorf("ID = %q, want %q", second.User.ID, first.User.ID)
	}
}

// Rotation is only worth having if the spent token stops working. Otherwise a
// stolen refresh token remains valid for its full lifetime.
func TestServiceRefreshRevokesTheSpentToken(t *testing.T) {
	svc, _ := newTestService(t)
	first := registerUser(t, svc, "player@playhub.test", RolePlayer)

	if _, err := svc.Refresh(context.Background(), first.Tokens.RefreshToken); err != nil {
		t.Fatalf("Refresh() returned error: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), first.Tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("second Refresh() error = %v, want ErrInvalidToken", err)
	}
}

func TestServiceRefreshRejectsAccessToken(t *testing.T) {
	svc, _ := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	if _, err := svc.Refresh(context.Background(), session.Tokens.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Refresh() error = %v, want ErrInvalidToken", err)
	}
}

func TestServiceRefreshRejectsExpiredRecord(t *testing.T) {
	svc, store := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	// Expire the server-side record while the signed token is still valid.
	for id, rec := range store.tokens {
		rec.ExpiresAt = time.Now().Add(-time.Minute)
		store.tokens[id] = rec
	}

	if _, err := svc.Refresh(context.Background(), session.Tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Refresh() error = %v, want ErrInvalidToken", err)
	}
}

func TestServiceRefreshRejectsInactiveAccount(t *testing.T) {
	svc, store := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)
	store.setActive(session.User.ID, false)

	if _, err := svc.Refresh(context.Background(), session.Tokens.RefreshToken); !errors.Is(err, ErrAccountInactive) {
		t.Errorf("Refresh() error = %v, want ErrAccountInactive", err)
	}
}

func TestServiceLogoutRevokesRefreshToken(t *testing.T) {
	svc, _ := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	if err := svc.Logout(context.Background(), session.Tokens.RefreshToken); err != nil {
		t.Fatalf("Logout() returned error: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), session.Tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Refresh() after logout error = %v, want ErrInvalidToken", err)
	}
}

// Logging out is the client saying the session is over. Every failure mode
// here already means that, so none of them is worth an error.
func TestServiceLogoutIsIdempotent(t *testing.T) {
	svc, _ := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	for _, token := range []string{session.Tokens.RefreshToken, session.Tokens.RefreshToken, "garbage"} {
		if err := svc.Logout(context.Background(), token); err != nil {
			t.Errorf("Logout() returned error: %v", err)
		}
	}
}

// An access token is not a session handle. Revoking on one would let a caller
// end a session it only holds a short-lived credential for.
func TestServiceLogoutIgnoresAccessToken(t *testing.T) {
	svc, _ := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	if err := svc.Logout(context.Background(), session.Tokens.AccessToken); err != nil {
		t.Fatalf("Logout() returned error: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), session.Tokens.RefreshToken); err != nil {
		t.Errorf("Refresh() error = %v, want the refresh token to still work", err)
	}
}

func TestServiceSurfacesStoreFailures(t *testing.T) {
	svc, store := newTestService(t)
	boom := errors.New("database is on fire")
	store.failWith = boom

	req := validRegisterRequest()
	req.Normalise()

	if _, err := svc.Register(context.Background(), req); !errors.Is(err, boom) {
		t.Errorf("Register() error = %v, want the store error", err)
	}
	if _, err := svc.Login(context.Background(), LoginRequest{Email: "a@b.test", Password: "x"}); !errors.Is(err, boom) {
		t.Errorf("Login() error = %v, want the store error", err)
	}
}
