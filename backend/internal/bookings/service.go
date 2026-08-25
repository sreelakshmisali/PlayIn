package bookings

import "context"

// Store is the persistence the service needs. It is declared here, as an
// interface, so the service can be tested without PostgreSQL and so the
// dependency points inward: *Repository satisfies it, not the other way
// round.
type Store interface {
	CreateBooking(ctx context.Context, playerID, slotID string) (Booking, error)
	BookingsForPlayer(ctx context.Context, playerID string) ([]Booking, error)
	BookingByID(ctx context.Context, playerID, bookingID string) (Booking, error)
	CancelBooking(ctx context.Context, playerID, bookingID string) (Booking, error)
}

// Service holds the booking rules. There are deliberately few: which slot is
// bookable, and which of a player's own bookings is theirs to see or cancel,
// are both enforced by the Store itself (the guarded reservation, and every
// query being scoped to playerID), not re-checked here on a copy the service
// read earlier.
type Service struct {
	store Store
}

// NewService wires a Service from its dependencies.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateBooking reserves req's slot for playerID.
func (s *Service) CreateBooking(ctx context.Context, playerID string, req CreateBookingRequest) (Booking, error) {
	return s.store.CreateBooking(ctx, playerID, req.TurfSlotID)
}

// MyBookings lists playerID's own bookings, most recent first.
func (s *Service) MyBookings(ctx context.Context, playerID string) ([]Booking, error) {
	return s.store.BookingsForPlayer(ctx, playerID)
}

// Booking reads one of playerID's own bookings.
func (s *Service) Booking(ctx context.Context, playerID, bookingID string) (Booking, error) {
	return s.store.BookingByID(ctx, playerID, bookingID)
}

// CancelBooking moves one of playerID's own bookings from CONFIRMED to
// CANCELLED and frees its slot.
func (s *Service) CancelBooking(ctx context.Context, playerID, bookingID string) (Booking, error) {
	return s.store.CancelBooking(ctx, playerID, bookingID)
}
