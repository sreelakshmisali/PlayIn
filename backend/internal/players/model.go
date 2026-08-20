// Package players owns player profiles and the sports catalogue they draw on.
//
// It follows the shape auth established: service.go holds the rules behind a
// narrow Store interface, repository.go is the only place that writes SQL, and
// handler.go translates HTTP without deciding anything.
//
// The sports catalogue lives here rather than in a package of its own. It is a
// single read-only table that exists to be referenced by a preferred sport;
// splitting it off would create a package seam with nothing on either side of
// it.
package players

import (
	"errors"
	"fmt"
	"time"
)

// Sport is one entry in the catalogue.
//
// Positions is the closed set a player may choose for this sport. It is empty
// for sports that have no positions, which is how "where applicable" is
// expressed: the client offers a position picker when the list is non-empty and
// omits it when it is not.
type Sport struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Positions []string  `json:"positions"`
	CreatedAt time.Time `json:"-"`
}

// HasPosition reports whether position is one this sport offers.
func (s Sport) HasPosition(position string) bool {
	for _, p := range s.Positions {
		if p == position {
			return true
		}
	}
	return false
}

// PlayerSport is a sport a player prefers, with the position they play in it.
// Position is empty for sports that have none, and may be empty for sports that
// do: choosing a sport does not oblige a player to name a position.
type PlayerSport struct {
	Sport    Sport  `json:"sport"`
	Position string `json:"position,omitempty"`
}

// Profile is a player's public sports profile.
//
// This is the only shape the package ever serialises, and it is deliberately
// built from the profile tables alone. No field here comes from users, so no
// email address, password hash, role or account flag can reach a response
// through it, whether the reader is the owner or a stranger.
//
// The profile's own primary key is not exposed either. Callers address a
// player by user id, which is the identifier the rest of the API already uses.
type Profile struct {
	UserID      string        `json:"user_id"`
	DisplayName string        `json:"display_name"`
	ImageURL    string        `json:"image_url,omitempty"`
	Bio         string        `json:"bio,omitempty"`
	Location    string        `json:"location,omitempty"`
	Sports      []PlayerSport `json:"sports"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// profileRow is the stored profile, including the surrogate key the repository
// needs to join player_sports. It never leaves the package.
type profileRow struct {
	ID          string
	UserID      string
	DisplayName string
	ImageURL    string
	Bio         string
	Location    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// toProfile projects a stored row and its sports into the response shape.
func (p profileRow) toProfile(sports []PlayerSport) Profile {
	if sports == nil {
		// A player with no preferred sports serialises as [], not null, so the
		// client never has to guard the array before iterating it.
		sports = []PlayerSport{}
	}

	return Profile{
		UserID:      p.UserID,
		DisplayName: p.DisplayName,
		ImageURL:    p.ImageURL,
		Bio:         p.Bio,
		Location:    p.Location,
		Sports:      sports,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// profileFields are the writable columns of a profile. Empty strings mean the
// column is NULL; display_name is the only one that cannot be empty.
type profileFields struct {
	DisplayName string
	ImageURL    string
	Bio         string
	Location    string
}

// Errors returned by the service. Handlers map these to status codes; nothing
// downstream branches on error text.
var (
	// ErrProfileNotFound means the user has no profile yet, or no such user.
	ErrProfileNotFound = errors.New("players: profile not found")
	// ErrSportNotFound means the sport id is unknown or the sport is retired.
	ErrSportNotFound = errors.New("players: sport not found")
	// ErrInvalidPosition means the position is not one the sport offers.
	ErrInvalidPosition = errors.New("players: invalid position for sport")
	// ErrSportNotPreferred means the player has not chosen that sport.
	ErrSportNotPreferred = errors.New("players: sport is not a preferred sport")
)

// PositionError reports a position the sport does not offer. It carries the
// sport so the handler can tell the client what the valid choices are, which is
// the difference between a usable error and a guessing game.
type PositionError struct {
	Sport Sport
}

func (e *PositionError) Error() string {
	if len(e.Sport.Positions) == 0 {
		return fmt.Sprintf("players: %s has no positions", e.Sport.Name)
	}
	return fmt.Sprintf("players: invalid position for %s", e.Sport.Name)
}

// Unwrap lets callers match with errors.Is(err, ErrInvalidPosition).
func (e *PositionError) Unwrap() error { return ErrInvalidPosition }

// Message is the client-facing explanation, listing the accepted values.
func (e *PositionError) Message() string {
	if len(e.Sport.Positions) == 0 {
		return e.Sport.Name + " does not use positions."
	}
	return "Position must be one of: " + joinWithCommas(e.Sport.Positions) + "."
}

func joinWithCommas(values []string) string {
	out := ""
	for i, v := range values {
		switch {
		case i == 0:
			out = v
		case i == len(values)-1:
			out += " or " + v
		default:
			out += ", " + v
		}
	}
	return out
}
