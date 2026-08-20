// Package owners owns owner profiles and the turfs they list.
//
// It follows the shape auth and players established: service.go holds the
// rules behind a narrow Store interface, repository.go is the only place that
// writes SQL, and handler.go translates HTTP without deciding anything.
//
// Turfs live in this package rather than a package of their own. A turf is
// entirely an owner-owned concern the same way a player's preferred sports are
// entirely a player-owned concern in the players package, and splitting it off
// would create a package seam with nothing on either side of it. The amenities
// catalogue lives here for the same reason the sports catalogue lives in
// players: a small read-only table that exists to be referenced by one join.
package owners

import (
	"errors"
	"time"
)

// Profile is an owner's business profile.
//
// This is the only shape the package ever serialises for an owner profile, and
// it is built from the profile table alone. No field here comes from users, so
// no email address, password hash, role or account flag can reach a response
// through it.
type Profile struct {
	UserID      string    `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Phone       string    `json:"phone,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// profileRow is the stored profile, including the surrogate key the repository
// needs to join turfs. It never leaves the package.
type profileRow struct {
	ID          string
	UserID      string
	DisplayName string
	Phone       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (p profileRow) toProfile() Profile {
	return Profile{
		UserID:      p.UserID,
		DisplayName: p.DisplayName,
		Phone:       p.Phone,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// profileFields are the writable columns of an owner profile. Empty strings
// mean the column is NULL; DisplayName is the only one that cannot be empty.
type profileFields struct {
	DisplayName string
	Phone       string
	Description string
}

// Errors returned by the service. Handlers map these to status codes; nothing
// downstream branches on error text.
var (
	// ErrOwnerProfileNotFound means the user has no owner profile yet.
	ErrOwnerProfileNotFound = errors.New("owners: profile not found")
)
