package owners

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mustDate parses an ISO date for a test fixture. The layout is fixed and
// under the test's own control, so a parse failure is a broken test, not a
// runtime condition to handle.
func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	d, err := time.Parse(dateLayout, value)
	if err != nil {
		t.Fatalf("parsing fixture date %q failed: %v", value, err)
	}
	return d
}

// createTestTurfWithSlotSettings creates a turf (06:00-22:00) with a
// configured 60-minute, price-500 slot setting, ready for slot generation.
func createTestTurfWithSlotSettings(t *testing.T, repo *Repository, ownerProfileID string) Turf {
	t.Helper()

	turf, err := repo.CreateTurf(context.Background(), ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	updated, err := repo.UpdateSlotSettings(context.Background(), ownerProfileID, turf.ID, slotSettingsFields{DurationMinutes: 60, Price: 500})
	if err != nil {
		t.Fatalf("UpdateSlotSettings() returned error: %v", err)
	}
	return updated
}

func TestRepositoryUpdateSlotSettings(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)

	if turf.SlotDurationMinutes == nil || *turf.SlotDurationMinutes != 60 {
		t.Errorf("SlotDurationMinutes = %v, want 60", turf.SlotDurationMinutes)
	}
	if turf.SlotPrice == nil || *turf.SlotPrice != 500 {
		t.Errorf("SlotPrice = %v, want 500", turf.SlotPrice)
	}
}

func TestRepositoryInsertSlotsAndSlotsInRange(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
		{Date: mustDate(t, "2026-09-01"), StartTime: "11:00", EndTime: "12:00", Price: 500},
	})
	if err != nil {
		t.Fatalf("InsertSlots() returned error: %v", err)
	}

	slots, err := repo.SlotsInRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("len(slots) = %d, want 2", len(slots))
	}
	if !slots[0].Available || slots[0].Status != SlotStatusOpen {
		t.Errorf("slot 0 Status/Available = %v/%v, want OPEN/true", slots[0].Status, slots[0].Available)
	}
}

// The exclusion constraint, not just a same-start-time uniqueness check, is
// what InsertSlots relies on: a candidate that overlaps an existing slot at a
// *different* start time must still be silently skipped.
func TestRepositoryInsertSlotsSkipsGenuineOverlap(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
	}); err != nil {
		t.Fatalf("first InsertSlots() returned error: %v", err)
	}

	// 10:30-11:30 overlaps the existing 10:00-11:00 slot without sharing its
	// start time.
	if err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:30", EndTime: "11:30", Price: 500},
	}); err != nil {
		t.Fatalf("overlapping InsertSlots() returned error: %v, want it silently skipped, not errored", err)
	}

	slots, err := repo.SlotsInRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("len(slots) = %d, want 1 (the overlapping candidate must not have been inserted)", len(slots))
	}
}

func TestRepositoryPublicSlotsForDateRequiresApproved(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
	}); err != nil {
		t.Fatalf("InsertSlots() returned error: %v", err)
	}

	if _, err := repo.PublicSlotsForDate(ctx, turf.ID, mustDate(t, "2026-09-01")); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("PublicSlotsForDate() on a DRAFT turf error = %v, want ErrTurfNotFound", err)
	}

	if _, err := testPool.Exec(ctx, `UPDATE turfs SET status = 'APPROVED' WHERE id = $1`, turf.ID); err != nil {
		t.Fatalf("approving the turf failed: %v", err)
	}

	slots, err := repo.PublicSlotsForDate(ctx, turf.ID, mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("PublicSlotsForDate() returned error: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("len(slots) = %d, want 1", len(slots))
	}
}

func TestRepositorySetSlotStatus(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
	}); err != nil {
		t.Fatalf("InsertSlots() returned error: %v", err)
	}
	slots, err := repo.SlotsInRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}

	blocked, err := repo.SetSlotStatus(ctx, ownerProfileID, turf.ID, slots[0].ID, SlotStatusBlocked)
	if err != nil {
		t.Fatalf("SetSlotStatus() returned error: %v", err)
	}
	if blocked.Status != SlotStatusBlocked || blocked.Available {
		t.Errorf("Status/Available = %v/%v, want BLOCKED/false", blocked.Status, blocked.Available)
	}
}

func TestRepositorySetSlotStatusNotFound(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)

	if _, err := repo.SetSlotStatus(context.Background(), ownerProfileID, turf.ID, "00000000-0000-0000-0000-000000000000", SlotStatusBlocked); !errors.Is(err, ErrSlotNotFound) {
		t.Errorf("SetSlotStatus() error = %v, want ErrSlotNotFound", err)
	}
}

func TestRepositoryDeleteSlot(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
	}); err != nil {
		t.Fatalf("InsertSlots() returned error: %v", err)
	}
	slots, err := repo.SlotsInRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}

	if err := repo.DeleteSlot(ctx, ownerProfileID, turf.ID, slots[0].ID); err != nil {
		t.Fatalf("DeleteSlot() returned error: %v", err)
	}

	remaining, err := repo.SlotsInRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("len(remaining) = %d, want 0", len(remaining))
	}
}

// A blocked date makes an already-generated OPEN slot unavailable without any
// write to the slot row itself: the availability projection re-derives it on
// every read against the live block table.
func TestRepositoryBlockDateDrivesAvailability(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
	}); err != nil {
		t.Fatalf("InsertSlots() returned error: %v", err)
	}

	before, err := repo.SlotsInRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	if !before[0].Available {
		t.Fatal("slot is unavailable before any block exists")
	}

	if _, err := repo.BlockDate(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), "Holiday"); err != nil {
		t.Fatalf("BlockDate() returned error: %v", err)
	}

	after, err := repo.SlotsInRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01"))
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	if after[0].Available {
		t.Error("slot is still available after its date was blocked")
	}
	if after[0].Status != SlotStatusOpen {
		t.Errorf("Status = %v, want OPEN (blocking a date must not rewrite the slot row)", after[0].Status)
	}
}

func TestRepositoryBlockDateDuplicateRejected(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if _, err := repo.BlockDate(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), ""); err != nil {
		t.Fatalf("first BlockDate() returned error: %v", err)
	}
	if _, err := repo.BlockDate(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), ""); !errors.Is(err, ErrDateAlreadyBlocked) {
		t.Errorf("second BlockDate() error = %v, want ErrDateAlreadyBlocked", err)
	}
}

func TestRepositoryUnblockDate(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	blocked, err := repo.BlockDate(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), "Holiday")
	if err != nil {
		t.Fatalf("BlockDate() returned error: %v", err)
	}
	if err := repo.UnblockDate(ctx, ownerProfileID, turf.ID, blocked.ID); err != nil {
		t.Fatalf("UnblockDate() returned error: %v", err)
	}

	dates, err := repo.BlockedDates(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("BlockedDates() returned error: %v", err)
	}
	if len(dates) != 0 {
		t.Errorf("BlockedDates() = %d, want 0", len(dates))
	}
}

// The database-level half of the overlap guard: two blocked time ranges that
// genuinely overlap, at different start times, are refused by the
// turf_blocked_time_ranges_no_overlap exclusion constraint itself, not by
// anything the service pre-checked.
func TestRepositoryBlockTimeRangeExclusionConstraint(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if _, err := repo.BlockTimeRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), "10:00", "12:00", ""); err != nil {
		t.Fatalf("first BlockTimeRange() returned error: %v", err)
	}

	_, err := repo.BlockTimeRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), "11:00", "13:00", "")
	if !errors.Is(err, ErrTimeRangeOverlapsBlock) {
		t.Errorf("overlapping BlockTimeRange() error = %v, want ErrTimeRangeOverlapsBlock", err)
	}

	// Back-to-back, not overlapping: must succeed.
	if _, err := repo.BlockTimeRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), "12:00", "13:00", ""); err != nil {
		t.Errorf("back-to-back BlockTimeRange() returned error: %v, want success", err)
	}
}

func TestRepositoryUnblockTimeRange(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	blocked, err := repo.BlockTimeRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), "10:00", "12:00", "")
	if err != nil {
		t.Fatalf("BlockTimeRange() returned error: %v", err)
	}
	if err := repo.UnblockTimeRange(ctx, ownerProfileID, turf.ID, blocked.ID); err != nil {
		t.Fatalf("UnblockTimeRange() returned error: %v", err)
	}

	ranges, err := repo.BlockedTimeRanges(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("BlockedTimeRanges() returned error: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("BlockedTimeRanges() = %d, want 0", len(ranges))
	}
}

// Deleting a turf must take its slots and blocks with it: ON DELETE CASCADE,
// not an application-level cleanup step.
func TestRepositorySlotsAndBlocksCascadeWithTurf(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerProfileID)
	ctx := context.Background()

	if err := repo.InsertSlots(ctx, ownerProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
	}); err != nil {
		t.Fatalf("InsertSlots() returned error: %v", err)
	}
	if _, err := repo.BlockDate(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-02"), ""); err != nil {
		t.Fatalf("BlockDate() returned error: %v", err)
	}
	if _, err := repo.BlockTimeRange(ctx, ownerProfileID, turf.ID, mustDate(t, "2026-09-01"), "14:00", "15:00", ""); err != nil {
		t.Fatalf("BlockTimeRange() returned error: %v", err)
	}

	if err := repo.DeleteTurf(ctx, ownerProfileID, turf.ID); err != nil {
		t.Fatalf("DeleteTurf() returned error: %v", err)
	}

	var slotCount, blockedDateCount, blockedRangeCount int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM turf_slots WHERE turf_id = $1`, turf.ID).Scan(&slotCount); err != nil {
		t.Fatalf("counting turf_slots failed: %v", err)
	}
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM turf_blocked_dates WHERE turf_id = $1`, turf.ID).Scan(&blockedDateCount); err != nil {
		t.Fatalf("counting turf_blocked_dates failed: %v", err)
	}
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM turf_blocked_time_ranges WHERE turf_id = $1`, turf.ID).Scan(&blockedRangeCount); err != nil {
		t.Fatalf("counting turf_blocked_time_ranges failed: %v", err)
	}
	if slotCount != 0 || blockedDateCount != 0 || blockedRangeCount != 0 {
		t.Errorf("rows survived turf deletion: slots=%d blocked_dates=%d blocked_time_ranges=%d",
			slotCount, blockedDateCount, blockedRangeCount)
	}
}

// A second owner's turf id must not be reachable through any slot or block
// operation, the same isolation every other owner-scoped query enforces.
func TestRepositorySlotOwnershipIsolation(t *testing.T) {
	repo := newTestRepository(t)
	ownerAProfileID := createTestOwnerProfile(t, repo)
	turf := createTestTurfWithSlotSettings(t, repo, ownerAProfileID)
	ownerBProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	if err := repo.InsertSlots(ctx, ownerAProfileID, turf.ID, []slotCandidate{
		{Date: mustDate(t, "2026-09-01"), StartTime: "10:00", EndTime: "11:00", Price: 500},
	}); err != nil {
		t.Fatalf("InsertSlots() returned error: %v", err)
	}

	if _, err := repo.SlotsInRange(ctx, ownerBProfileID, turf.ID, mustDate(t, "2026-09-01"), mustDate(t, "2026-09-01")); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("SlotsInRange() by a different owner error = %v, want ErrTurfNotFound", err)
	}
	if _, err := repo.UpdateSlotSettings(ctx, ownerBProfileID, turf.ID, slotSettingsFields{DurationMinutes: 30, Price: 100}); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("UpdateSlotSettings() by a different owner error = %v, want ErrTurfNotFound", err)
	}
	if _, err := repo.BlockDate(ctx, ownerBProfileID, turf.ID, mustDate(t, "2026-09-05"), ""); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("BlockDate() by a different owner error = %v, want ErrTurfNotFound", err)
	}
}
