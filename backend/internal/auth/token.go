package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/orgmelethil/playhub/backend/internal/config"
)

// TokenKind separates the two token types that share a signing key.
//
// Without it a refresh token, which is long lived by design, would be accepted
// by the authentication middleware as a month-long access token. Every token
// carries its kind and every verifier states the kind it expects.
type TokenKind string

const (
	// KindAccess authorises API calls. Short lived, never stored server side.
	KindAccess TokenKind = "access"
	// KindRefresh buys a new pair. Its ID is a row in refresh_tokens, so it can
	// be revoked.
	KindRefresh TokenKind = "refresh"
)

// bearerScheme is the Authorization scheme the API accepts.
const bearerScheme = "Bearer"

// Claims is the token payload: the registered claims plus what the API needs
// to authorise a request without a database round trip.
type Claims struct {
	jwt.RegisteredClaims
	Kind TokenKind `json:"kind"`
	Role Role      `json:"role"`
}

// UserID returns the subject, which is the user's UUID.
func (c Claims) UserID() string { return c.Subject }

// Issuer mints and verifies tokens. One instance is built in cmd/api and shared.
type Issuer struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// NewIssuer wires an Issuer from configuration.
func NewIssuer(cfg config.Auth) *Issuer {
	return &Issuer{
		secret:     []byte(cfg.JWTSecret),
		issuer:     cfg.JWTIssuer,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		now:        time.Now,
	}
}

// AccessTTL exposes the configured access token lifetime, which the service
// reports to clients as expires_in.
func (i *Issuer) AccessTTL() time.Duration { return i.accessTTL }

// RefreshExpiry returns the absolute expiry a refresh token issued now will
// carry. The service writes it to refresh_tokens before signing the token, so
// the row and the claim always agree.
func (i *Issuer) RefreshExpiry() time.Time { return i.now().Add(i.refreshTTL).UTC() }

// Access signs an access token for the user.
func (i *Issuer) Access(user User) (string, error) {
	issued := i.now().UTC()
	return i.sign(Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(issued),
			NotBefore: jwt.NewNumericDate(issued),
			ExpiresAt: jwt.NewNumericDate(issued.Add(i.accessTTL)),
		},
		Kind: KindAccess,
		Role: user.Role,
	})
}

// Refresh signs a refresh token bound to an existing refresh_tokens row.
// recordID becomes the jti claim and expiresAt mirrors the row's expires_at.
func (i *Issuer) Refresh(user User, recordID string, expiresAt time.Time) (string, error) {
	issued := i.now().UTC()
	return i.sign(Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        recordID,
			Issuer:    i.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(issued),
			NotBefore: jwt.NewNumericDate(issued),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Kind: KindRefresh,
		Role: user.Role,
	})
}

// sign serialises and signs a claim set with HS256.
func (i *Issuer) sign(claims Claims) (string, error) {
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Parse verifies a token's signature, its registered claims and its kind.
// Every failure collapses to ErrInvalidToken: a caller holding a bad token has
// nothing useful to do with the distinction, and the detail would tell an
// attacker which part of the forgery to fix.
func (i *Issuer) Parse(token string, want TokenKind) (*Claims, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(token, claims,
		func(t *jwt.Token) (any, error) {
			// Pinning the method here is what stops the alg-confusion attack,
			// where a token re-signed as "none" or as RS256 with the public key
			// would otherwise verify.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
			}
			return i.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(i.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(i.now),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims.Kind != want {
		return nil, fmt.Errorf("%w: token is a %s token, want %s", ErrInvalidToken, claims.Kind, want)
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: subject is empty", ErrInvalidToken)
	}
	if want == KindRefresh && claims.ID == "" {
		return nil, fmt.Errorf("%w: refresh token has no id", ErrInvalidToken)
	}

	return claims, nil
}
