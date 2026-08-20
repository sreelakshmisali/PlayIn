// Package auth owns user accounts, credentials and session tokens.
//
// It follows the layering the health package established: service.go holds the
// logic behind narrow interfaces, repository.go is the only place that touches
// SQL, and handler.go translates HTTP without deciding anything.
package auth

import (
	"errors"
	"time"
)

// Role is a user's authorisation level. The set is closed and mirrors the
// users_role_chk constraint in migration 000002.
type Role string

const (
	// RolePlayer books turfs and joins teams. It is the default on signup.
	RolePlayer Role = "PLAYER"
	// RoleOwner manages turfs, slots and their bookings.
	RoleOwner Role = "OWNER"
	// RoleAdmin administers the platform. It is never self-assignable.
	RoleAdmin Role = "ADMIN"
)

// Valid reports whether r is one of the known roles.
func (r Role) Valid() bool {
	switch r {
	case RolePlayer, RoleOwner, RoleAdmin:
		return true
	default:
		return false
	}
}

// SelfAssignable reports whether a user may choose this role when registering.
// ADMIN is granted out of band, never by the account holder.
func (r Role) SelfAssignable() bool {
	return r == RolePlayer || r == RoleOwner
}

// User is an account. PasswordHash is deliberately unexported from JSON: this
// struct is the repository's row type and must never be serialised to a client.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Role         Role
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Profile is the client-facing view of a user. It is a separate type rather
// than json tags on User so a hash can never leak through a forgotten tag.
type Profile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      Role      `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Profile returns the safe projection of u.
func (u User) Profile() Profile {
	return Profile{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}

// TokenPair is what a successful register, login or refresh returns.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	// ExpiresIn is the access token lifetime in seconds, so a client can
	// schedule a refresh without decoding the token.
	ExpiresIn int64 `json:"expires_in"`
}

// Session is the full response body for register, login and refresh.
type Session struct {
	User   Profile   `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

// RefreshRecord is a server-side record of one issued refresh token. Its ID is
// the token's jti claim, which is what makes revocation possible.
type RefreshRecord struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Usable reports whether the record still authorises a refresh at now.
func (r RefreshRecord) Usable(now time.Time) bool {
	return r.RevokedAt == nil && now.Before(r.ExpiresAt)
}

// Errors returned by the service. Handlers map these to status codes; nothing
// downstream branches on error text.
var (
	// ErrEmailTaken means an account already exists for the address.
	ErrEmailTaken = errors.New("auth: email already registered")
	// ErrInvalidCredentials covers both an unknown email and a wrong password.
	// They share one error so the API cannot be used to enumerate accounts.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrAccountInactive means the account exists but has been disabled.
	ErrAccountInactive = errors.New("auth: account is inactive")
	// ErrInvalidToken covers a malformed, expired, revoked or unknown token.
	ErrInvalidToken = errors.New("auth: invalid token")
	// ErrUserNotFound means the subject of a valid token no longer exists.
	ErrUserNotFound = errors.New("auth: user not found")
	// ErrCannotModifySelf means an admin action targeted the caller's own
	// account. Deactivating yourself is not a normal moderation outcome, it is
	// almost always a mistake, and Authenticate re-checks IsActive on every
	// request, so it would lock the admin out immediately, including from
	// undoing it.
	ErrCannotModifySelf = errors.New("auth: cannot perform this action on your own account")
)
