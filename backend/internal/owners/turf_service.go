package owners

import "context"

// Amenities returns the amenities catalogue an owner chooses from.
func (s *Service) Amenities(ctx context.Context) ([]Amenity, error) {
	return s.store.Amenities(ctx)
}

// CreateTurf lists a new turf under the caller's owner profile.
//
// It requires the profile to exist first, the same anchor players' preferred
// sports use against a player profile: a turf needs somewhere to say who runs
// it, and ErrOwnerProfileNotFound tells the client exactly what is missing
// rather than failing on a foreign key it never sees.
func (s *Service) CreateTurf(ctx context.Context, userID string, req SaveTurfRequest) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	return s.store.CreateTurf(ctx, ownerProfileID, req.fields())
}

// MyTurfs lists every turf the caller owns, any status.
func (s *Service) MyTurfs(ctx context.Context, userID string) ([]Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.store.TurfsByOwner(ctx, ownerProfileID)
}

// MyTurf reads one of the caller's own turfs.
func (s *Service) MyTurf(ctx context.Context, userID, turfID string) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	return s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// UpdateTurf replaces the details of one of the caller's own turfs.
func (s *Service) UpdateTurf(ctx context.Context, userID, turfID string, req SaveTurfRequest) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	return s.store.UpdateTurf(ctx, ownerProfileID, turfID, req.fields())
}

// DeleteTurf removes one of the caller's own turfs.
func (s *Service) DeleteTurf(ctx context.Context, userID, turfID string) error {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.store.DeleteTurf(ctx, ownerProfileID, turfID)
}

// SubmitTurf asks for review, moving a DRAFT or REJECTED turf to
// PENDING_APPROVAL. Nothing in this phase moves it out of that status; that is
// the admin surface Phase 4 adds.
func (s *Service) SubmitTurf(ctx context.Context, userID, turfID string) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	return s.store.SubmitTurf(ctx, ownerProfileID, turfID)
}

// SetTurfSport attaches a sport to one of the caller's own turfs.
func (s *Service) SetTurfSport(ctx context.Context, userID, turfID string, req SetTurfSportRequest) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	if err := s.store.SetTurfSport(ctx, ownerProfileID, turfID, req.SportID); err != nil {
		return Turf{}, err
	}
	return s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// RemoveTurfSport detaches a sport from one of the caller's own turfs.
func (s *Service) RemoveTurfSport(ctx context.Context, userID, turfID, sportID string) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	if err := s.store.RemoveTurfSport(ctx, ownerProfileID, turfID, sportID); err != nil {
		return Turf{}, err
	}
	return s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// SetTurfAmenity attaches an amenity to one of the caller's own turfs.
func (s *Service) SetTurfAmenity(ctx context.Context, userID, turfID string, req SetTurfAmenityRequest) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	if err := s.store.SetTurfAmenity(ctx, ownerProfileID, turfID, req.AmenityID); err != nil {
		return Turf{}, err
	}
	return s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// RemoveTurfAmenity detaches an amenity from one of the caller's own turfs.
func (s *Service) RemoveTurfAmenity(ctx context.Context, userID, turfID, amenityID string) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	if err := s.store.RemoveTurfAmenity(ctx, ownerProfileID, turfID, amenityID); err != nil {
		return Turf{}, err
	}
	return s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// AddTurfImage appends an image URL to one of the caller's own turfs.
func (s *Service) AddTurfImage(ctx context.Context, userID, turfID string, req AddTurfImageRequest) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	if _, err := s.store.AddTurfImage(ctx, ownerProfileID, turfID, req.ImageURL); err != nil {
		return Turf{}, err
	}
	return s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// RemoveTurfImage drops an image from one of the caller's own turfs.
func (s *Service) RemoveTurfImage(ctx context.Context, userID, turfID, imageID string) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	if err := s.store.RemoveTurfImage(ctx, ownerProfileID, turfID, imageID); err != nil {
		return Turf{}, err
	}
	return s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// PublicTurfs lists APPROVED turfs for anyone to browse.
func (s *Service) PublicTurfs(ctx context.Context) ([]Turf, error) {
	return s.store.PublicTurfs(ctx)
}

// PublicTurf reads one APPROVED turf. Anything else, including a turf that
// simply does not exist, reports ErrTurfNotFound.
func (s *Service) PublicTurf(ctx context.Context, turfID string) (Turf, error) {
	return s.store.PublicTurfByID(ctx, turfID)
}
