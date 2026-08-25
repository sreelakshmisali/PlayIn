package bookings

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository tests run against a real PostgreSQL, because what they check is
// the schema: the guarded reservation UPDATE, the partial unique index, the
// composite foreign key and the CHECK constraints. A fake would only re-test
// the fake — see support_test.go's memStore for the unit-level rules these
// same scenarios exercise without a database.
//
// They are skipped unless PLAYHUB_TEST_DATABASE_URL points at a migrated
// database, so `go test ./...` stays runnable without one.
const testDatabaseURLEnv = "PLAYHUB_TEST_DATABASE_URL"

var (
	testPoolOnce sync.Once
	testPool     *pgxpool.Pool
	testPoolErr  error
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()

	dsn := os.Getenv(testDatabaseURLEnv)
	if dsn == "" {
		t.Skipf("%s is not set, skipping the repository tests", testDatabaseURLEnv)
	}

	testPoolOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		testPool, testPoolErr = pgxpool.New(ctx, dsn)
		if testPoolErr == nil {
			testPoolErr = testPool.Ping(ctx)
		}
	})
	if testPoolErr != nil {
		t.Fatalf("connecting to the test database failed: %v", testPoolErr)
	}

	return NewRepository(testPool)
}

func sanitise(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}

func uniqueEmail(t *testing.T, tag string) string {
	t.Helper()
	return fmt.Sprintf("%s-%s-%d@playhub.test", sanitise(t.Name()), tag, time.Now().UnixNano())
}

// createTestPlayer inserts a PLAYER account and removes it (cascading
// through its bookings) when the test ends.
func createTestPlayer(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	var userID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, '$2a$04$notarealhashbutlongenough', 'Test Player', 'PLAYER')
		RETURNING id::text`, uniqueEmail(t, "player")).Scan(&userID)
	if err != nil {
		t.Fatalf("creating the test player failed: %v", err)
	}

	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("cleaning up player %s failed: %v", userID, err)
		}
	})
	return userID
}

// testSlot is what a test needs to address a seeded slot: its own id, and
// the turf it belongs to (for a test that reaches past the bookings package
// to change that turf's state directly).
type testSlot struct {
	TurfID string
	SlotID string
}

// seedBookableSlot creates an OWNER account, an owner profile, an APPROVED
// turf and one OPEN turf_slot on it — a slot CreateBooking should be able to
// reserve. Deleting the owner account cascades through everything else.
func seedBookableSlot(t *testing.T) testSlot {
	t.Helper()
	return seedSlotWithStatus(t, "OPEN")
}

// seedSlotWithStatus is seedBookableSlot with the slot's own status set to
// something other than OPEN (e.g. BLOCKED), for tests of a slot that should
// not be bookable for that reason.
func seedSlotWithStatus(t *testing.T, status string) testSlot {
	t.Helper()

	ctx := context.Background()

	var ownerUserID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, '$2a$04$notarealhashbutlongenough', 'Test Owner', 'OWNER')
		RETURNING id::text`, uniqueEmail(t, "owner")).Scan(&ownerUserID)
	if err != nil {
		t.Fatalf("creating the test owner failed: %v", err)
	}

	var ownerProfileID string
	err = testPool.QueryRow(ctx, `
		INSERT INTO owner_profiles (user_id, display_name)
		VALUES ($1, 'Test Arena')
		RETURNING id::text`, ownerUserID).Scan(&ownerProfileID)
	if err != nil {
		t.Fatalf("creating the test owner profile failed: %v", err)
	}

	var turfID string
	err = testPool.QueryRow(ctx, `
		INSERT INTO turfs (owner_id, name, address, city, opening_time, closing_time, status)
		VALUES ($1, 'Test Turf', '123 Test Road', 'Kochi', '06:00', '22:00', 'APPROVED')
		RETURNING id::text`, ownerProfileID).Scan(&turfID)
	if err != nil {
		t.Fatalf("creating the test turf failed: %v", err)
	}

	var slotID string
	err = testPool.QueryRow(ctx, `
		INSERT INTO turf_slots (turf_id, slot_date, start_time, end_time, price, status)
		VALUES ($1, CURRENT_DATE + 1, '18:00', '19:00', 500, $2)
		RETURNING id::text`, turfID, status).Scan(&slotID)
	if err != nil {
		t.Fatalf("creating the test slot failed: %v", err)
	}

	// Registered after the turf and slot exist so it runs before them (t.Cleanup
	// is LIFO). bookings_slot_fk is ON DELETE RESTRICT by design — booking
	// history outlives the slot it points at — so a test's own bookings have to
	// be cleared explicitly before deleting the owner cascades the slot away.
	t.Cleanup(func() {
		ctx := context.Background()
		if _, err := testPool.Exec(ctx, `DELETE FROM bookings WHERE turf_id = $1`, turfID); err != nil {
			t.Errorf("cleaning up bookings for turf %s failed: %v", turfID, err)
		}
		if _, err := testPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, ownerUserID); err != nil {
			t.Errorf("cleaning up owner %s failed: %v", ownerUserID, err)
		}
	})

	return testSlot{TurfID: turfID, SlotID: slotID}
}

// blockSlotDate blocks the whole date a seeded slot falls on, so the slot is
// OPEN but not available.
func blockSlotDate(t *testing.T, turfID string) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `
		INSERT INTO turf_blocked_dates (turf_id, blocked_date)
		VALUES ($1, CURRENT_DATE + 1)`, turfID)
	if err != nil {
		t.Fatalf("blocking the test slot's date failed: %v", err)
	}
}

func slotStatus(t *testing.T, slotID string) string {
	t.Helper()
	var status string
	if err := testPool.QueryRow(context.Background(), `SELECT status FROM turf_slots WHERE id = $1`, slotID).Scan(&status); err != nil {
		t.Fatalf("reading slot status failed: %v", err)
	}
	return status
}

// --- successful booking -----------------------------------------------------

func TestRepositoryCreateBookingSucceeds(t *testing.T) {
	repo := newTestRepository(t)
	playerID := createTestPlayer(t)
	slot := seedBookableSlot(t)

	booking, err := repo.CreateBooking(context.Background(), playerID, slot.SlotID)
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}

	if booking.TurfID != slot.TurfID {
		t.Errorf("TurfID = %q, want %q", booking.TurfID, slot.TurfID)
	}
	if booking.TurfSlotID != slot.SlotID {
		t.Errorf("TurfSlotID = %q, want %q", booking.TurfSlotID, slot.SlotID)
	}
	if booking.Status != StatusConfirmed {
		t.Errorf("Status = %q, want %q", booking.Status, StatusConfirmed)
	}
	if booking.Price != 500 {
		t.Errorf("Price = %v, want 500", booking.Price)
	}
	if booking.StartTime != "18:00" || booking.EndTime != "19:00" {
		t.Errorf("StartTime/EndTime = %s/%s, want 18:00/19:00", booking.StartTime, booking.EndTime)
	}

	// The slot itself is now BOOKED.
	if got := slotStatus(t, slot.SlotID); got != "BOOKED" {
		t.Errorf("slot status = %q, want BOOKED", got)
	}
}

// --- duplicate booking of the same slot --------------------------------------

func TestRepositoryCreateBookingRefusesDuplicateOnSameSlot(t *testing.T) {
	repo := newTestRepository(t)
	playerA := createTestPlayer(t)
	playerB := createTestPlayer(t)
	slot := seedBookableSlot(t)

	if _, err := repo.CreateBooking(context.Background(), playerA, slot.SlotID); err != nil {
		t.Fatalf("first CreateBooking() returned error: %v", err)
	}

	_, err := repo.CreateBooking(context.Background(), playerB, slot.SlotID)
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("second CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}
}

// --- booking a non-OPEN slot --------------------------------------------------

func TestRepositoryCreateBookingRefusesNonOpenSlot(t *testing.T) {
	repo := newTestRepository(t)
	playerID := createTestPlayer(t)
	slot := seedSlotWithStatus(t, "BLOCKED")

	_, err := repo.CreateBooking(context.Background(), playerID, slot.SlotID)
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}
}

// --- booking an unavailable/blocked slot -------------------------------------
//
// Distinct from the above: the slot's own status is OPEN, but its date is
// blocked outright, the same "available" computation turf_slots' own read
// projection performs.

func TestRepositoryCreateBookingRefusesSlotOnABlockedDate(t *testing.T) {
	repo := newTestRepository(t)
	playerID := createTestPlayer(t)
	slot := seedBookableSlot(t)
	blockSlotDate(t, slot.TurfID)

	_, err := repo.CreateBooking(context.Background(), playerID, slot.SlotID)
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}

	// The slot's own status is untouched: a blocked date, not a status
	// change, is why the booking was refused.
	if got := slotStatus(t, slot.SlotID); got != "OPEN" {
		t.Errorf("slot status = %q, want OPEN", got)
	}
}

func TestRepositoryCreateBookingRefusesUnknownSlot(t *testing.T) {
	repo := newTestRepository(t)
	playerID := createTestPlayer(t)

	_, err := repo.CreateBooking(context.Background(), playerID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrSlotNotBookable) {
		t.Errorf("CreateBooking() error = %v, want ErrSlotNotBookable", err)
	}
}

// --- player isolation ---------------------------------------------------------

func TestRepositoryBookingIsIsolatedToItsOwnPlayer(t *testing.T) {
	repo := newTestRepository(t)
	owner := createTestPlayer(t)
	stranger := createTestPlayer(t)
	slot := seedBookableSlot(t)

	booking, err := repo.CreateBooking(context.Background(), owner, slot.SlotID)
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}

	if _, err := repo.BookingByID(context.Background(), stranger, booking.ID); !errors.Is(err, ErrBookingNotFound) {
		t.Errorf("BookingByID() by a different player error = %v, want ErrBookingNotFound", err)
	}
	if _, err := repo.CancelBooking(context.Background(), stranger, booking.ID); !errors.Is(err, ErrBookingNotFound) {
		t.Errorf("CancelBooking() by a different player error = %v, want ErrBookingNotFound", err)
	}

	list, err := repo.BookingsForPlayer(context.Background(), stranger)
	if err != nil {
		t.Fatalf("BookingsForPlayer() returned error: %v", err)
	}
	for _, b := range list {
		if b.ID == booking.ID {
			t.Errorf("BookingsForPlayer(stranger) contains another player's booking %s", booking.ID)
		}
	}
}

// --- cancellation --------------------------------------------------------------

func TestRepositoryCancelBookingSucceedsAndFreesTheSlot(t *testing.T) {
	repo := newTestRepository(t)
	playerID := createTestPlayer(t)
	slot := seedBookableSlot(t)

	booking, err := repo.CreateBooking(context.Background(), playerID, slot.SlotID)
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}

	cancelled, err := repo.CancelBooking(context.Background(), playerID, booking.ID)
	if err != nil {
		t.Fatalf("CancelBooking() returned error: %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Errorf("Status = %q, want %q", cancelled.Status, StatusCancelled)
	}
	if cancelled.CancelledAt == nil {
		t.Error("CancelledAt = nil, want a timestamp")
	}

	if got := slotStatus(t, slot.SlotID); got != "OPEN" {
		t.Errorf("slot status after cancel = %q, want OPEN", got)
	}

	// The freed slot can be booked again, by anyone.
	otherPlayer := createTestPlayer(t)
	if _, err := repo.CreateBooking(context.Background(), otherPlayer, slot.SlotID); err != nil {
		t.Errorf("re-booking the freed slot returned error: %v", err)
	}
}

// --- invalid state transitions -------------------------------------------------

func TestRepositoryCancelBookingRefusesSecondCancel(t *testing.T) {
	repo := newTestRepository(t)
	playerID := createTestPlayer(t)
	slot := seedBookableSlot(t)

	booking, err := repo.CreateBooking(context.Background(), playerID, slot.SlotID)
	if err != nil {
		t.Fatalf("CreateBooking() returned error: %v", err)
	}
	if _, err := repo.CancelBooking(context.Background(), playerID, booking.ID); err != nil {
		t.Fatalf("first CancelBooking() returned error: %v", err)
	}

	_, err = repo.CancelBooking(context.Background(), playerID, booking.ID)
	if !errors.Is(err, ErrAlreadyCancelled) {
		t.Errorf("second CancelBooking() error = %v, want ErrAlreadyCancelled", err)
	}
}

func TestRepositoryCancelBookingRefusesUnknownBooking(t *testing.T) {
	repo := newTestRepository(t)
	playerID := createTestPlayer(t)

	_, err := repo.CancelBooking(context.Background(), playerID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrBookingNotFound) {
		t.Errorf("CancelBooking() error = %v, want ErrBookingNotFound", err)
	}
}

// --- concurrent booking attempts ------------------------------------------------
//
// The one property that cannot be proven by a fake or by sequential calls: N
// goroutines racing to book the same slot through N separate connections
// from the pool must produce exactly one CONFIRMED booking, never more. This
// is what migration 000007's guarded UPDATE plus the
// bookings_slot_confirmed_key partial unique index exist for.

func TestRepositoryCreateBookingIsConcurrencySafe(t *testing.T) {
	repo := newTestRepository(t)
	slot := seedBookableSlot(t)

	const attempts = 10
	players := make([]string, attempts)
	for i := range players {
		players[i] = createTestPlayer(t)
	}

	var wg sync.WaitGroup
	results := make([]error, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.CreateBooking(context.Background(), players[i], slot.SlotID)
			results[i] = err
		}(i)
	}
	wg.Wait()

	successes, refusals := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrSlotNotBookable):
			refusals++
		default:
			t.Errorf("CreateBooking() returned an unexpected error: %v", err)
		}
	}

	if successes != 1 {
		t.Errorf("successes = %d, want exactly 1", successes)
	}
	if successes+refusals != attempts {
		t.Errorf("successes+refusals = %d, want %d", successes+refusals, attempts)
	}

	// The database agrees: exactly one CONFIRMED booking exists for the slot.
	var confirmedCount int
	err := testPool.QueryRow(context.Background(), `
		SELECT count(*) FROM bookings WHERE turf_slot_id = $1 AND status = 'CONFIRMED'`,
		slot.SlotID).Scan(&confirmedCount)
	if err != nil {
		t.Fatalf("counting confirmed bookings failed: %v", err)
	}
	if confirmedCount != 1 {
		t.Errorf("confirmed bookings in the database = %d, want 1", confirmedCount)
	}
}
