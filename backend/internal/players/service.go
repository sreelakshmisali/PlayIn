package players

import (
	"context"
	"errors"
)

// Store is the persistence the service needs. It is declared here so the
// dependency points inward: *Repository satisfies it, not the other way round,
// and the rules stay testable without PostgreSQL.
type Store interface {
	Sports(ctx context.Context) ([]Sport, error)
	SportByID(ctx context.Context, id string) (Sport, error)
	ProfileByUserID(ctx context.Context, userID string) (Profile, error)
	ProfileIDForUser(ctx context.Context, userID string) (string, error)
	SaveProfile(ctx context.Context, userID string, f profileFields) (Profile, bool, error)
	SetSport(ctx context.Context, profileID, sportID, position string) error
	RemoveSport(ctx context.Context, profileID, sportID string) error
}

// Service holds the player profile rules.
type Service struct {
	store Store
}

// NewService wires a Service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Sports returns the catalogue a player chooses from.
func (s *Service) Sports(ctx context.Context) ([]Sport, error) {
	return s.store.Sports(ctx)
}

// Profile reads one player's profile. It is the same call whether the reader is
// the owner or a stranger, because the profile carries nothing an owner should
// see and a stranger should not.
func (s *Service) Profile(ctx context.Context, userID string) (Profile, error) {
	return s.store.ProfileByUserID(ctx, userID)
}

// SaveProfile creates or replaces a profile. The second return reports whether
// it was created, which the handler answers with 201 instead of 200.
func (s *Service) SaveProfile(ctx context.Context, userID string, req SaveProfileRequest) (Profile, bool, error) {
	return s.store.SaveProfile(ctx, userID, req.fields())
}

// PatchProfile applies a partial update to an existing profile.
//
// It reads first so an absent field keeps its stored value, and refuses when
// there is no profile: patching implies something to patch. PUT is the verb
// that creates one.
func (s *Service) PatchProfile(ctx context.Context, userID string, req PatchProfileRequest) (Profile, error) {
	current, err := s.store.ProfileByUserID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	merged := req.apply(profileFields{
		DisplayName: current.DisplayName,
		ImageURL:    current.ImageURL,
		Bio:         current.Bio,
		Location:    current.Location,
	})

	updated, _, err := s.store.SaveProfile(ctx, userID, merged)
	if err != nil {
		return Profile{}, err
	}
	return updated, nil
}

// SetSport adds a preferred sport, or changes the position on one the player
// already has. It returns the profile so a client gets the new sports list
// without a follow-up read.
//
// The write itself enforces that the sport exists, is active, and offers the
// position. Only when it refuses does this read the sport, and only to say
// which of the three was wrong.
func (s *Service) SetSport(ctx context.Context, userID string, req SetSportRequest) (Profile, error) {
	profileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	if err := s.store.SetSport(ctx, profileID, req.SportID, req.Position); err != nil {
		if errors.Is(err, ErrSportNotFound) {
			return Profile{}, s.explainRejection(ctx, req)
		}
		return Profile{}, err
	}

	return s.store.ProfileByUserID(ctx, userID)
}

// RemoveSport drops a preferred sport and returns the profile without it.
func (s *Service) RemoveSport(ctx context.Context, userID, sportID string) (Profile, error) {
	profileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}

	if err := s.store.RemoveSport(ctx, profileID, sportID); err != nil {
		return Profile{}, err
	}

	return s.store.ProfileByUserID(ctx, userID)
}

// explainRejection turns a refused write into an error a client can act on.
// The sport is unknown, or it exists and does not offer the position.
func (s *Service) explainRejection(ctx context.Context, req SetSportRequest) error {
	sport, err := s.store.SportByID(ctx, req.SportID)
	if err != nil {
		// Including a genuine storage failure: if the sport cannot be read, the
		// most useful thing to say is still that it could not be used.
		return ErrSportNotFound
	}
	if req.Position != "" && !sport.HasPosition(req.Position) {
		return &PositionError{Sport: sport}
	}
	return ErrSportNotFound
}
