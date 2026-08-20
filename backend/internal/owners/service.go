package owners

import "context"

// Store is the persistence the service needs. It is declared here, as an
// interface, so the service can be tested without PostgreSQL and so the
// dependency points inward: *Repository satisfies it, not the other way round.
type Store interface {
	ProfileByUserID(ctx context.Context, userID string) (Profile, error)
	ProfileIDForUser(ctx context.Context, userID string) (string, error)
	SaveProfile(ctx context.Context, userID string, f profileFields) (Profile, bool, error)

	CreateTurf(ctx context.Context, ownerProfileID string, f turfFields) (Turf, error)
	TurfsByOwner(ctx context.Context, ownerProfileID string) ([]Turf, error)
	TurfByOwnerAndID(ctx context.Context, ownerProfileID, turfID string) (Turf, error)
	UpdateTurf(ctx context.Context, ownerProfileID, turfID string, f turfFields) (Turf, error)
	DeleteTurf(ctx context.Context, ownerProfileID, turfID string) error
	SubmitTurf(ctx context.Context, ownerProfileID, turfID string) (Turf, error)

	SetTurfSport(ctx context.Context, ownerProfileID, turfID, sportID string) error
	RemoveTurfSport(ctx context.Context, ownerProfileID, turfID, sportID string) error
	SetTurfAmenity(ctx context.Context, ownerProfileID, turfID, amenityID string) error
	RemoveTurfAmenity(ctx context.Context, ownerProfileID, turfID, amenityID string) error
	AddTurfImage(ctx context.Context, ownerProfileID, turfID, imageURL string) (TurfImage, error)
	RemoveTurfImage(ctx context.Context, ownerProfileID, turfID, imageID string) error

	Amenities(ctx context.Context) ([]Amenity, error)
	PublicTurfs(ctx context.Context) ([]Turf, error)
	PublicTurfByID(ctx context.Context, turfID string) (Turf, error)

	PendingTurfs(ctx context.Context) ([]Turf, error)
	TurfByID(ctx context.Context, turfID string) (Turf, error)
	ApproveTurf(ctx context.Context, turfID, adminUserID string) (Turf, error)
	RejectTurf(ctx context.Context, turfID, adminUserID, reason string) (Turf, error)
	SuspendTurf(ctx context.Context, turfID, adminUserID, reason string) (Turf, error)
	RestoreTurf(ctx context.Context, turfID, adminUserID string) (Turf, error)
}

// Service holds the owner profile and turf rules.
type Service struct {
	store Store
}

// NewService wires a Service from its dependencies.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Profile reads an owner's profile.
func (s *Service) Profile(ctx context.Context, userID string) (Profile, error) {
	return s.store.ProfileByUserID(ctx, userID)
}

// SaveProfile creates or replaces a profile. The second return reports
// whether it was created, which the handler answers with 201 instead of 200.
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
		Phone:       current.Phone,
		Description: current.Description,
	})

	updated, _, err := s.store.SaveProfile(ctx, userID, merged)
	if err != nil {
		return Profile{}, err
	}
	return updated, nil
}
