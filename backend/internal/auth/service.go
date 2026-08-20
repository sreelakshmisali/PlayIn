package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// Store is the persistence the service needs. It is declared here, as an
// interface, so the service can be tested without PostgreSQL and so the
// dependency points inward: *Repository satisfies it, not the other way round.
type Store interface {
	CreateUser(ctx context.Context, email, passwordHash, fullName string, role Role) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	UserByID(ctx context.Context, id string) (User, error)
	CreateRefreshToken(ctx context.Context, userID string, expiresAt time.Time) (string, error)
	RefreshTokenByID(ctx context.Context, id string) (RefreshRecord, error)
	RevokeRefreshToken(ctx context.Context, id string) error

	Users(ctx context.Context, limit, offset int) ([]User, error)
	UserCount(ctx context.Context) (int, error)
	SetActive(ctx context.Context, userID string, active bool) (User, error)
}

// Service holds the authentication rules.
type Service struct {
	store  Store
	hasher Hasher
	tokens *Issuer
	// decoyHash is verified against when a login names an unknown email, so
	// that path costs the same as a wrong password. See Login.
	decoyHash string
}

// NewService wires a Service from its dependencies.
func NewService(store Store, hasher Hasher, tokens *Issuer) *Service {
	return &Service{
		store:     store,
		hasher:    hasher,
		tokens:    tokens,
		decoyHash: newDecoyHash(hasher),
	}
}

// newDecoyHash hashes a random string with the configured hasher, once, at
// startup. Deriving it here rather than hard-coding a constant keeps its work
// factor equal to the one real password hashes use, which is the whole point.
func newDecoyHash(hasher Hasher) string {
	buf := make([]byte, 32)
	// crypto/rand.Read never returns an error as of Go 1.24.
	_, _ = rand.Read(buf)

	hash, err := hasher.Hash(base64.RawStdEncoding.EncodeToString(buf))
	if err != nil {
		// A hasher that cannot hash is a startup problem that surfaces on the
		// first real request. Logins still work; only the timing cover is lost.
		return ""
	}
	return hash
}

// Register creates an account and signs the caller straight in.
//
// The request is expected to be normalised and validated by the caller; the
// unique index on users.email is what actually settles a race between two
// simultaneous signups for the same address.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (Session, error) {
	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return Session{}, err
	}

	user, err := s.store.CreateUser(ctx, req.Email, hash, req.FullName, req.Role)
	if err != nil {
		return Session{}, err
	}

	return s.issue(ctx, user)
}

// Login exchanges credentials for a token pair.
//
// An unknown email still costs a password verification. Returning early would
// make a missing account measurably faster than a wrong password and turn the
// endpoint into an account enumeration oracle.
func (s *Service) Login(ctx context.Context, req LoginRequest) (Session, error) {
	user, err := s.store.UserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = s.hasher.Verify(s.decoyHash, req.Password)
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, err
	}

	if err := s.hasher.Verify(user.PasswordHash, req.Password); err != nil {
		return Session{}, err
	}
	if !user.IsActive {
		return Session{}, ErrAccountInactive
	}

	return s.issue(ctx, user)
}

// Refresh rotates a refresh token for a new pair.
//
// The presented token is revoked before the replacement is issued, so a stolen
// token is single use: whichever party redeems it first invalidates it for the
// other.
func (s *Service) Refresh(ctx context.Context, token string) (Session, error) {
	claims, err := s.tokens.Parse(token, KindRefresh)
	if err != nil {
		return Session{}, err
	}

	record, err := s.store.RefreshTokenByID(ctx, claims.ID)
	if err != nil {
		return Session{}, err
	}
	if !record.Usable(time.Now()) || record.UserID != claims.Subject {
		return Session{}, ErrInvalidToken
	}

	user, err := s.store.UserByID(ctx, record.UserID)
	if err != nil {
		return Session{}, err
	}
	if !user.IsActive {
		return Session{}, ErrAccountInactive
	}

	if err := s.store.RevokeRefreshToken(ctx, record.ID); err != nil {
		return Session{}, err
	}

	return s.issue(ctx, user)
}

// Logout revokes a refresh token.
//
// A token that is malformed, expired or already revoked is treated as success.
// The caller's intent is to end a session, and every one of those states means
// the session is already over; reporting an error would only invite a client to
// retry something that cannot be improved.
func (s *Service) Logout(ctx context.Context, token string) error {
	claims, err := s.tokens.Parse(token, KindRefresh)
	if err != nil {
		return nil
	}
	if err := s.store.RevokeRefreshToken(ctx, claims.ID); err != nil {
		return err
	}
	return nil
}

// Authenticate verifies an access token and returns the account behind it.
//
// It reads the user on every call rather than trusting the claims, so a
// deactivated or deleted account stops working immediately instead of at the
// end of the access token's lifetime.
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	claims, err := s.tokens.Parse(token, KindAccess)
	if err != nil {
		return User{}, err
	}

	user, err := s.store.UserByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, ErrInvalidToken
		}
		return User{}, err
	}
	if !user.IsActive {
		return User{}, ErrAccountInactive
	}

	return user, nil
}

// issue records a refresh token and signs both halves of the pair. The record
// is written first so the jti in the signed token always has a row behind it.
func (s *Service) issue(ctx context.Context, user User) (Session, error) {
	expiresAt := s.tokens.RefreshExpiry()

	recordID, err := s.store.CreateRefreshToken(ctx, user.ID, expiresAt)
	if err != nil {
		return Session{}, err
	}

	access, err := s.tokens.Access(user)
	if err != nil {
		return Session{}, fmt.Errorf("sign access token: %w", err)
	}

	refresh, err := s.tokens.Refresh(user, recordID, expiresAt)
	if err != nil {
		return Session{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return Session{
		User: user.Profile(),
		Tokens: TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
			TokenType:    bearerScheme,
			ExpiresIn:    int64(s.tokens.AccessTTL().Seconds()),
		},
	}, nil
}
