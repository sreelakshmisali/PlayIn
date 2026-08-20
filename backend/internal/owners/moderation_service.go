package owners

import "context"

// PendingTurfs lists turfs awaiting admin review.
func (s *Service) PendingTurfs(ctx context.Context) ([]Turf, error) {
	return s.store.PendingTurfs(ctx)
}

// AdminTurf reads any turf by id, regardless of owner or status.
func (s *Service) AdminTurf(ctx context.Context, turfID string) (Turf, error) {
	return s.store.TurfByID(ctx, turfID)
}

// ApproveTurf publishes a submitted turf.
func (s *Service) ApproveTurf(ctx context.Context, turfID, adminUserID string) (Turf, error) {
	return s.store.ApproveTurf(ctx, turfID, adminUserID)
}

// RejectTurf declines a submitted turf with a reason the owner can act on.
func (s *Service) RejectTurf(ctx context.Context, turfID, adminUserID string, req ModerateTurfRequest) (Turf, error) {
	return s.store.RejectTurf(ctx, turfID, adminUserID, req.Reason)
}

// SuspendTurf pulls a live turf out of public visibility with a reason.
func (s *Service) SuspendTurf(ctx context.Context, turfID, adminUserID string, req ModerateTurfRequest) (Turf, error) {
	return s.store.SuspendTurf(ctx, turfID, adminUserID, req.Reason)
}

// RestoreTurf reinstates a suspended turf.
func (s *Service) RestoreTurf(ctx context.Context, turfID, adminUserID string) (Turf, error) {
	return s.store.RestoreTurf(ctx, turfID, adminUserID)
}
