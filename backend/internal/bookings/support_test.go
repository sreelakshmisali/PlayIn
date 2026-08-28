package bookings

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/orgmelethil/playhub/backend/internal/auth"
)

// fakeSlotState is the minimal stand-in for a turf_slots row's own
// bookability: whether the service's booking rules should currently be able
// to reserve it. It intentionally does not model OPEN/BLOCKED/BOOKED as
// three independently-meaningful values the way the real schema does —
// only "reservable" (open) or not — because the fake exists to exercise the
// booking rules the Store interface exposes, not to re-implement
// turf_slots' own availability computation, which repository_test.go
// exercises against the real schema instead.
type fakeSlotState string

const (
	fakeSlotOpen    fakeSlotState = "OPEN"    // reservable
	fakeSlotBlocked fakeSlotState = "BLOCKED" // owner-blocked or inside a blocked date/range
	fakeSlotBooked  fakeSlotState = "BOOKED"  // already reserved by a CONFIRMED booking
)

type fakeSlot struct {
	ID        string
	TurfID    string
	Date      string
	StartTime string
	EndTime   string
	Price     float64
	State     fakeSlotState
}

// memStore is an in-memory Store. It is single-threaded-correct, not
// concurrency-safe the way the real repository is proven to be in
// repository_test.go — a fake would only re-test the fake for that. What it
// does verify is the booking rules themselves: a slot must be OPEN to be
// reserved, at most one CONFIRMED booking exists per slot, a booking is
// scoped to the player who made it, and cancellation is a one-way,
// one-shot transition that frees the slot back up.
type memStore struct {
	mu       sync.Mutex
	slots    map[string]*fakeSlot
	bookings map[string]*bookingRow
	seq      int
	failWith error
}

func newMemStore() *memStore {
	return &memStore{
		slots:    make(map[string]*fakeSlot),
		bookings: make(map[string]*bookingRow),
	}
}

func (m *memStore) nextID(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s-%d", prefix, m.seq)
}

// seedSlot adds a fake slot in the given state and returns its id. Callers
// hold no lock; seedSlot takes its own.
func (m *memStore) seedSlot(state fakeSlotState) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID("slot")
	m.slots[id] = &fakeSlot{
		ID: id, TurfID: "turf-1", Date: "2026-09-01",
		StartTime: "18:00", EndTime: "19:00", Price: 500,
		State: state,
	}
	return id
}

func (m *memStore) CreateBooking(_ context.Context, playerID, slotID string) (Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Booking{}, m.failWith
	}

	slot, ok := m.slots[slotID]
	if !ok || slot.State != fakeSlotOpen {
		return Booking{}, ErrSlotNotBookable
	}

	slot.State = fakeSlotBooked

	now := time.Now().UTC()
	date, err := time.Parse(dateLayout, slot.Date)
	if err != nil {
		return Booking{}, fmt.Errorf("parse fake slot date: %w", err)
	}

	id := m.nextID("booking")
	row := &bookingRow{
		ID: id, PlayerID: playerID, TurfID: slot.TurfID, TurfSlotID: slot.ID,
		Date: date, StartTime: slot.StartTime, EndTime: slot.EndTime,
		Status: StatusConfirmed, Price: slot.Price,
		CreatedAt: now, UpdatedAt: now,
	}
	m.bookings[id] = row
	return row.toBooking(), nil
}

func (m *memStore) BookingsForPlayer(_ context.Context, playerID string) ([]Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}

	out := make([]Booking, 0, len(m.bookings))
	for _, row := range m.bookings {
		if row.PlayerID == playerID {
			out = append(out, row.toBooking())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *memStore) BookingByID(_ context.Context, playerID, bookingID string) (Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Booking{}, m.failWith
	}

	row, ok := m.bookings[bookingID]
	if !ok || row.PlayerID != playerID {
		return Booking{}, ErrBookingNotFound
	}
	return row.toBooking(), nil
}

func (m *memStore) CancelBooking(_ context.Context, playerID, bookingID string) (Booking, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Booking{}, m.failWith
	}

	row, ok := m.bookings[bookingID]
	if !ok || row.PlayerID != playerID {
		return Booking{}, ErrBookingNotFound
	}
	if row.Status != StatusConfirmed {
		return Booking{}, ErrAlreadyCancelled
	}

	now := time.Now().UTC()
	row.Status = StatusCancelled
	row.CancelledAt = &now
	row.UpdatedAt = now

	if slot, ok := m.slots[row.TurfSlotID]; ok {
		slot.State = fakeSlotOpen
	}
	return row.toBooking(), nil
}

func newTestService() (*Service, *memStore) {
	store := newMemStore()
	return NewService(store), store
}

// stubAuthenticator resolves any bearer token to a fixed user, so the
// handler tests exercise the real guards without minting real JWTs.
type stubAuthenticator struct {
	user auth.User
	err  error
}

func (s stubAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return s.user, s.err
}

func testUser(role auth.Role) auth.User {
	return auth.User{ID: testPlayerID, Email: "player@playhub.test", Role: role, IsActive: true}
}

func newTestHandler(user auth.User) (*Handler, *memStore) {
	svc, store := newTestService()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(svc, stubAuthenticator{user: user}, logger), store
}
