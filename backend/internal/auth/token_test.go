package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testUser() User {
	return User{ID: "user-1", Email: "a@playhub.test", Role: RolePlayer, IsActive: true}
}

func TestIssuerAccessRoundTrip(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())
	user := testUser()

	token, err := issuer.Access(user)
	if err != nil {
		t.Fatalf("Access() returned error: %v", err)
	}

	claims, err := issuer.Parse(token, KindAccess)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if claims.UserID() != user.ID {
		t.Errorf("UserID() = %q, want %q", claims.UserID(), user.ID)
	}
	if claims.Role != RolePlayer {
		t.Errorf("Role = %q, want %q", claims.Role, RolePlayer)
	}
	if claims.Issuer != "playhub-test" {
		t.Errorf("Issuer = %q, want playhub-test", claims.Issuer)
	}
}

func TestIssuerRefreshCarriesRecordID(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())
	expiry := issuer.RefreshExpiry()

	token, err := issuer.Refresh(testUser(), "rt-9", expiry)
	if err != nil {
		t.Fatalf("Refresh() returned error: %v", err)
	}

	claims, err := issuer.Parse(token, KindRefresh)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if claims.ID != "rt-9" {
		t.Errorf("ID = %q, want rt-9", claims.ID)
	}
	if got := claims.ExpiresAt.Time.Unix(); got != expiry.Unix() {
		t.Errorf("ExpiresAt = %d, want %d", got, expiry.Unix())
	}
}

// A refresh token is long lived. Accepting one where an access token is
// expected would silently extend every session to the refresh lifetime.
func TestIssuerRejectsWrongKind(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())

	refresh, err := issuer.Refresh(testUser(), "rt-1", issuer.RefreshExpiry())
	if err != nil {
		t.Fatalf("Refresh() returned error: %v", err)
	}

	if _, err := issuer.Parse(refresh, KindAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse(refresh, access) error = %v, want ErrInvalidToken", err)
	}
}

func TestIssuerRejectsForeignSecret(t *testing.T) {
	mine := NewIssuer(testAuthConfig())

	otherCfg := testAuthConfig()
	otherCfg.JWTSecret = strings.Repeat("z", 32)
	theirs := NewIssuer(otherCfg)

	token, err := theirs.Access(testUser())
	if err != nil {
		t.Fatalf("Access() returned error: %v", err)
	}

	if _, err := mine.Parse(token, KindAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestIssuerRejectsWrongIssuer(t *testing.T) {
	cfg := testAuthConfig()
	cfg.JWTIssuer = "somewhere-else"

	token, err := NewIssuer(cfg).Access(testUser())
	if err != nil {
		t.Fatalf("Access() returned error: %v", err)
	}

	if _, err := NewIssuer(testAuthConfig()).Parse(token, KindAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestIssuerRejectsExpiredToken(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())

	// Sign in the past so the token is already expired when it is parsed.
	past := time.Now().Add(-2 * time.Hour)
	issuer.now = func() time.Time { return past }

	token, err := issuer.Access(testUser())
	if err != nil {
		t.Fatalf("Access() returned error: %v", err)
	}

	issuer.now = time.Now
	if _, err := issuer.Parse(token, KindAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

// The alg-confusion attack: re-sign the payload with the none algorithm and
// hope the verifier trusts the header. Parse pins HS256, so it must not.
func TestIssuerRejectsUnsignedToken(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "playhub-test",
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Kind: KindAccess,
		Role: RoleAdmin,
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing the none-alg token failed: %v", err)
	}

	if _, err := issuer.Parse(unsigned, KindAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestIssuerRejectsMalformedToken(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())

	for _, token := range []string{"", "not-a-token", "a.b.c"} {
		if _, err := issuer.Parse(token, KindAccess); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalidToken", token, err)
		}
	}
}

func TestIssuerRejectsRefreshWithoutID(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())

	token, err := issuer.sign(Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "playhub-test",
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Kind: KindRefresh,
	})
	if err != nil {
		t.Fatalf("sign() returned error: %v", err)
	}

	if _, err := issuer.Parse(token, KindRefresh); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestIssuerRejectsEmptySubject(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())

	token, err := issuer.sign(Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "playhub-test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Kind: KindAccess,
	})
	if err != nil {
		t.Fatalf("sign() returned error: %v", err)
	}

	if _, err := issuer.Parse(token, KindAccess); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("Parse() error = %v, want ErrInvalidToken", err)
	}
}

func TestIssuerReportsTTLs(t *testing.T) {
	issuer := NewIssuer(testAuthConfig())

	if got := issuer.AccessTTL(); got != 15*time.Minute {
		t.Errorf("AccessTTL() = %v, want 15m", got)
	}
	if got := time.Until(issuer.RefreshExpiry()); got < 23*time.Hour {
		t.Errorf("RefreshExpiry() is %v away, want about 24h", got)
	}
}
