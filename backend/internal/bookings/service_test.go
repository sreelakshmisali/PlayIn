package bookings

import (
	"context"
	"errors"
	"testing"
)

const (
	testPlayerID  = "player-1"
	otherPlayerID = "player-2"
)

func TestServiceCreateBookingSucceeds(t *testing.T) {
	svc, store := newTestService()
	slotID := store.seedSlot(fakeSlotOpen)

	booking, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}

	if booking.TurfSlotID != slotID {
		t.Errorf("TurfSlotID = %q, want %q", booking.TurfSlotID, slotID)
	}
	if booking.Status != StatusConfirmed {
		t.Errorf("Status = %q, want %q", booking.Status, StatusConfirmed)
	}
	if booking.Price != 500 {
		t.Errorf("Price = %v, want 500", booking.Price)
	}
	if booking.CancelledAt != nil {
		t.Errorf("CancelledAt = %v, want nil", booking.CancelledAt)
	}
}

// A second booking attempt on a slot a first call already reserved must be
// refused — the same rule the guarded UPDATE and the partial unique index
// enforce structurally in repository_test.go.
func TestServiceCreateBookingRefusesDuplicateOnSameSlot(t *testing.T) {
	svc, store := newTestService()
	slotID := store.seedSlot(fakeSlotOpen)

	if _, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: slotID}); err != nil {
		t.Fatalf("first CreateBooking() returned error: %v", err)
	}

	_, err := svc.CreateBooking(context.Background(), otherPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("second CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}
}

func TestServiceCreateBookingRefusesNonOpenSlot(t *testing.T) {
	svc, store := newTestService()
	slotID := store.seedSlot(fakeSlotBooked)

	_, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}
}

func TestServiceCreateBookingRefusesBlockedSlot(t *testing.T) {
	svc, store := newTestService()
	slotID := store.seedSlot(fakeSlotBlocked)

	_, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}
}

func TestServiceCreateBookingRefusesUnknownSlot(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: "does-not-exist"})
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}
}

func TestServiceMyBookingsOnlyListsTheCallersOwn(t *testing.T) {
	svc, store := newTestService()
	mine := store.seedSlot(fakeSlotOpen)
	theirs := store.seedSlot(fakeSlotOpen)

	if _, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: mine}); err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}
	if _, err := svc.CreateBooking(context.Background(), otherPlayerID, CreateBookingRequest{TurfSlotID: theirs}); err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}

	list, err := svc.MyBookings(context.Background(), testPlayerID)
	if err != nil {
		t.Fatalf("MyBookings() returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].TurfSlotID != mine {
		t.Errorf("TurfSlotID = %q, want %q", list[0].TurfSlotID, mine)
	}
}

// Player isolation: a booking made by one player is invisible to another,
// both for reading and for cancelling it.
func TestServiceBookingIsIsolatedToItsOwnPlayer(t *testing.T) {
	svc, store := newTestService()
	slotID := store.seedSlot(fakeSlotOpen)

	booking, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}

	if _, err := svc.Booking(context.Background(), otherPlayerID, booking.ID); !errors.Is(err, ErrBookingNotFound) {
		t.Errorf("Booking() by a different player error = %v, want ErrBookingNotFound", err)
	}
	if _, err := svc.CancelBooking(context.Background(), otherPlayerID, booking.ID); !errors.Is(err, ErrBookingNotFound) {
		t.Errorf("CancelBooking() by a different player error = %v, want ErrBookingNotFound", err)
	}

	// The rightful owner can still see it, unaffected by the other player's
	// refused attempt.
	if _, err := svc.Booking(context.Background(), testPlayerID, booking.ID); err != nil {
		t.Errorf("Booking() by the owner returned error: %v", err)
	}
}

func TestServiceCancelBookingSucceedsAndFreesTheSlot(t *testing.T) {
	svc, store := newTestService()
	slotID := store.seedSlot(fakeSlotOpen)

	booking, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}

	cancelled, err := svc.CancelBooking(context.Background(), testPlayerID, booking.ID)
	if err != nil {
		t.Fatalf("CancelBooking() returned error: %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Errorf("Status = %q, want %q", cancelled.Status, StatusCancelled)
	}
	if cancelled.CancelledAt == nil {
		t.Error("CancelledAt = nil, want a timestamp")
	}

	// The slot is bookable again.
	rebooked, err := svc.CreateBooking(context.Background(), otherPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if err != nil {
		t.Fatalf("re-booking the freed slot returned error: %v", err)
	}
	if rebooked.Status != StatusConfirmed {
		t.Errorf("rebooked Status = %q, want %q", rebooked.Status, StatusConfirmed)
	}
}

// Cancelling a booking that is already cancelled is not a valid transition.
func TestServiceCancelBookingRefusesInvalidTransition(t *testing.T) {
	svc, store := newTestService()
	slotID := store.seedSlot(fakeSlotOpen)

	booking, err := svc.CreateBooking(context.Background(), testPlayerID, CreateBookingRequest{TurfSlotID: slotID})
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}
	if _, err := svc.CancelBooking(context.Background(), testPlayerID, booking.ID); err != nil {
		t.Fatalf("first CancelBooking() returned error: %v", err)
	}

	_, err = svc.CancelBooking(context.Background(), testPlayerID, booking.ID)
	if !errors.Is(err, ErrAlreadyCancelled) {
		t.Errorf("second CancelBooking() error = %v, want ErrAlreadyCancelled", err)
	}
}

func TestServiceCancelBookingRefusesUnknownBooking(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.CancelBooking(context.Background(), testPlayerID, "does-not-exist")
	if !errors.Is(err, ErrBookingNotFound) {
		t.Errorf("CancelBooking() error = %v, want ErrBookingNotFound", err)
	}
}
