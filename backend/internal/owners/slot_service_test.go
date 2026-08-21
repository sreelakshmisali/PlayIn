package owners

import (
	"context"
	"errors"
	"testing"
)

const testUserID = "user-1"

// seedTurfWithSlotSettings creates a turf (opening 06:00, closing 22:00) and
// configures it with a 60-minute, price-500 slot setting, ready for
// generation.
func seedTurfWithSlotSettings(t *testing.T, svc *Service, store *memStore) Turf {
	t.Helper()

	turf := seedTurf(t, svc, store, testUserID)
	updated, err := svc.UpdateSlotSettings(context.Background(), testUserID, turf.ID, SlotSettingsRequest{
		SlotDurationMinutes: 60,
		SlotPrice:           500,
	})
	if err != nil {
		t.Fatalf("UpdateSlotSettings() returned error: %v", err)
	}
	return updated
}

func TestServiceUpdateSlotSettings(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	if turf.SlotDurationMinutes == nil || *turf.SlotDurationMinutes != 60 {
		t.Fatalf("SlotDurationMinutes = %v, want 60", turf.SlotDurationMinutes)
	}
	if turf.SlotPrice == nil || *turf.SlotPrice != 500 {
		t.Fatalf("SlotPrice = %v, want 500", turf.SlotPrice)
	}
}

func TestServiceGenerateSlotsRequiresSettings(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, testUserID) // no slot settings configured

	_, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"})
	if !errors.Is(err, ErrSlotSettingsNotConfigured) {
		t.Errorf("GenerateSlots() error = %v, want ErrSlotSettingsNotConfigured", err)
	}
}

// A 06:00-22:00 window (960 minutes) at 60-minute slots produces exactly 16
// slots per day, the deterministic non-overlapping grid the design relies on.
func TestServiceGenerateSlotsBuildsTheGrid(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	slots, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}
	if len(slots) != 16 {
		t.Fatalf("len(slots) = %d, want 16", len(slots))
	}
	if slots[0].StartTime != "06:00" || slots[0].EndTime != "07:00" {
		t.Errorf("first slot = %s-%s, want 06:00-07:00", slots[0].StartTime, slots[0].EndTime)
	}
	if last := slots[len(slots)-1]; last.StartTime != "21:00" || last.EndTime != "22:00" {
		t.Errorf("last slot = %s-%s, want 21:00-22:00", last.StartTime, last.EndTime)
	}
	for _, s := range slots {
		if s.Price != 500 {
			t.Errorf("slot %s price = %v, want 500", s.StartTime, s.Price)
		}
		if s.Status != SlotStatusOpen || !s.Available {
			t.Errorf("slot %s status/available = %v/%v, want OPEN/true", s.StartTime, s.Status, s.Available)
		}
	}
}

// Multiple dates in one range each get their own full grid.
func TestServiceGenerateSlotsCoversEveryDateInRange(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	slots, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-03"})
	if err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}
	if len(slots) != 16*3 {
		t.Fatalf("len(slots) = %d, want %d across 3 dates", len(slots), 16*3)
	}

	byDate := map[string]int{}
	for _, s := range slots {
		byDate[s.Date]++
	}
	for _, date := range []string{"2026-09-01", "2026-09-02", "2026-09-03"} {
		if byDate[date] != 16 {
			t.Errorf("slots on %s = %d, want 16", date, byDate[date])
		}
	}
}

// Regenerating the same range is safe and does not duplicate rows: this is
// what lets an owner call generate again after changing a block.
func TestServiceGenerateSlotsIsIdempotent(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)
	req := GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"}

	first, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, req)
	if err != nil {
		t.Fatalf("first GenerateSlots() returned error: %v", err)
	}
	second, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, req)
	if err != nil {
		t.Fatalf("second GenerateSlots() returned error: %v", err)
	}
	if len(second) != len(first) {
		t.Errorf("len(second) = %d, want %d (no duplicates)", len(second), len(first))
	}
}

func TestServiceGenerateSlotsSkipsABlockedDate(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	if _, err := svc.BlockDate(context.Background(), testUserID, turf.ID, BlockDateRequest{Date: "2026-09-02", Reason: "Maintenance"}); err != nil {
		t.Fatalf("BlockDate() returned error: %v", err)
	}

	slots, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-03"})
	if err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}
	for _, s := range slots {
		if s.Date == "2026-09-02" {
			t.Fatalf("a slot was generated on the blocked date 2026-09-02: %+v", s)
		}
	}
	if len(slots) != 16*2 {
		t.Errorf("len(slots) = %d, want %d (2 open dates)", len(slots), 16*2)
	}
}

func TestServiceGenerateSlotsSkipsABlockedTimeRange(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	if _, err := svc.BlockTimeRange(context.Background(), testUserID, turf.ID, BlockTimeRangeRequest{
		Date: "2026-09-01", StartTime: "12:00", EndTime: "14:00", Reason: "Pitch resurfacing",
	}); err != nil {
		t.Fatalf("BlockTimeRange() returned error: %v", err)
	}

	slots, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}
	// 16 candidates minus the 12:00-13:00 and 13:00-14:00 slots the block covers.
	if len(slots) != 14 {
		t.Fatalf("len(slots) = %d, want 14", len(slots))
	}
	for _, s := range slots {
		if s.StartTime == "12:00" || s.StartTime == "13:00" {
			t.Errorf("a slot was generated inside the blocked range: %+v", s)
		}
	}
}

// A block added after slots already exist makes them unavailable immediately,
// without touching the slot rows: this is the "derive, don't duplicate"
// behaviour the design relies on.
func TestServiceBlockingAfterGenerationMakesSlotsUnavailable(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	slots, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}
	for _, s := range slots {
		if !s.Available {
			t.Fatalf("slot %s is unavailable before any block exists", s.StartTime)
		}
	}

	if _, err := svc.BlockDate(context.Background(), testUserID, turf.ID, BlockDateRequest{Date: "2026-09-01"}); err != nil {
		t.Fatalf("BlockDate() returned error: %v", err)
	}

	after, err := svc.SlotsInRange(context.Background(), testUserID, turf.ID, SlotRangeQuery{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	if len(after) != len(slots) {
		t.Fatalf("len(after) = %d, want %d (rows unchanged, only availability)", len(after), len(slots))
	}
	for _, s := range after {
		if s.Available {
			t.Errorf("slot %s is still available after its date was blocked", s.StartTime)
		}
		if s.Status != SlotStatusOpen {
			t.Errorf("slot %s Status = %v, want OPEN (blocking a date must not rewrite slot status)", s.StartTime, s.Status)
		}
	}
}

func TestServiceSetSlotStatusBlocksASingleSlot(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)
	slots, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}
	target := slots[0]

	blocked, err := svc.SetSlotStatus(context.Background(), testUserID, turf.ID, target.ID, SetSlotStatusRequest{Status: SlotStatusBlocked})
	if err != nil {
		t.Fatalf("SetSlotStatus() returned error: %v", err)
	}
	if blocked.Status != SlotStatusBlocked || blocked.Available {
		t.Errorf("Status/Available = %v/%v, want BLOCKED/false", blocked.Status, blocked.Available)
	}

	// The rest of the day is untouched.
	all, err := svc.SlotsInRange(context.Background(), testUserID, turf.ID, SlotRangeQuery{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	openCount := 0
	for _, s := range all {
		if s.Available {
			openCount++
		}
	}
	if openCount != len(all)-1 {
		t.Errorf("open count = %d, want %d (only the one slot blocked)", openCount, len(all)-1)
	}
}

func TestServiceDeleteSlot(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)
	slots, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}

	if err := svc.DeleteSlot(context.Background(), testUserID, turf.ID, slots[0].ID); err != nil {
		t.Fatalf("DeleteSlot() returned error: %v", err)
	}

	remaining, err := svc.SlotsInRange(context.Background(), testUserID, turf.ID, SlotRangeQuery{From: "2026-09-01", To: "2026-09-01"})
	if err != nil {
		t.Fatalf("SlotsInRange() returned error: %v", err)
	}
	if len(remaining) != len(slots)-1 {
		t.Errorf("len(remaining) = %d, want %d", len(remaining), len(slots)-1)
	}
}

func TestServiceDeleteSlotNotFound(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	if err := svc.DeleteSlot(context.Background(), testUserID, turf.ID, "slot-nobody"); !errors.Is(err, ErrSlotNotFound) {
		t.Errorf("DeleteSlot() error = %v, want ErrSlotNotFound", err)
	}
}

// A second owner's turf id must not be reachable through slot management,
// the same isolation every other owner-scoped turf operation enforces.
func TestServiceSlotsInRangeIsOwnerScoped(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)
	if _, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"}); err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}

	const otherUserID = "user-2"
	seedProfile(t, svc, otherUserID)

	if _, err := svc.SlotsInRange(context.Background(), otherUserID, turf.ID, SlotRangeQuery{From: "2026-09-01", To: "2026-09-01"}); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("SlotsInRange() by a different owner error = %v, want ErrTurfNotFound", err)
	}
}

func TestServicePublicSlotsForDateRequiresApproved(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)
	if _, err := svc.GenerateSlots(context.Background(), testUserID, turf.ID, GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"}); err != nil {
		t.Fatalf("GenerateSlots() returned error: %v", err)
	}

	// Still DRAFT: not publicly visible, same as PublicTurf.
	if _, err := svc.PublicSlotsForDate(context.Background(), turf.ID, "2026-09-01"); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("PublicSlotsForDate() on a DRAFT turf error = %v, want ErrTurfNotFound", err)
	}

	store.setStatus(turf.ID, StatusApproved)

	slots, err := svc.PublicSlotsForDate(context.Background(), turf.ID, "2026-09-01")
	if err != nil {
		t.Fatalf("PublicSlotsForDate() returned error: %v", err)
	}
	if len(slots) != 16 {
		t.Fatalf("len(slots) = %d, want 16", len(slots))
	}
}

func TestServiceBlockDateAlreadyBlocked(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	if _, err := svc.BlockDate(context.Background(), testUserID, turf.ID, BlockDateRequest{Date: "2026-09-01"}); err != nil {
		t.Fatalf("first BlockDate() returned error: %v", err)
	}
	if _, err := svc.BlockDate(context.Background(), testUserID, turf.ID, BlockDateRequest{Date: "2026-09-01"}); !errors.Is(err, ErrDateAlreadyBlocked) {
		t.Errorf("second BlockDate() error = %v, want ErrDateAlreadyBlocked", err)
	}
}

func TestServiceUnblockDate(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	blocked, err := svc.BlockDate(context.Background(), testUserID, turf.ID, BlockDateRequest{Date: "2026-09-01"})
	if err != nil {
		t.Fatalf("BlockDate() returned error: %v", err)
	}
	if err := svc.UnblockDate(context.Background(), testUserID, turf.ID, blocked.ID); err != nil {
		t.Fatalf("UnblockDate() returned error: %v", err)
	}

	dates, err := svc.BlockedDates(context.Background(), testUserID, turf.ID)
	if err != nil {
		t.Fatalf("BlockedDates() returned error: %v", err)
	}
	if len(dates) != 0 {
		t.Errorf("BlockedDates() = %d, want 0 after unblocking", len(dates))
	}
}

func TestServiceUnblockDateNotFound(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	if err := svc.UnblockDate(context.Background(), testUserID, turf.ID, "blocked-date-nobody"); !errors.Is(err, ErrBlockedDateNotFound) {
		t.Errorf("UnblockDate() error = %v, want ErrBlockedDateNotFound", err)
	}
}

// Two overlapping blocked time ranges on the same turf and date are refused,
// the service-level half of the overlap guard (the database's own EXCLUDE
// constraint is the other half, exercised in slot_repository_test.go).
func TestServiceBlockTimeRangeRejectsOverlap(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	if _, err := svc.BlockTimeRange(context.Background(), testUserID, turf.ID, BlockTimeRangeRequest{
		Date: "2026-09-01", StartTime: "10:00", EndTime: "12:00",
	}); err != nil {
		t.Fatalf("first BlockTimeRange() returned error: %v", err)
	}

	_, err := svc.BlockTimeRange(context.Background(), testUserID, turf.ID, BlockTimeRangeRequest{
		Date: "2026-09-01", StartTime: "11:00", EndTime: "13:00",
	})
	if !errors.Is(err, ErrTimeRangeOverlapsBlock) {
		t.Errorf("overlapping BlockTimeRange() error = %v, want ErrTimeRangeOverlapsBlock", err)
	}

	// Back-to-back, not overlapping, must succeed.
	if _, err := svc.BlockTimeRange(context.Background(), testUserID, turf.ID, BlockTimeRangeRequest{
		Date: "2026-09-01", StartTime: "12:00", EndTime: "13:00",
	}); err != nil {
		t.Errorf("back-to-back BlockTimeRange() returned error: %v, want success", err)
	}
}

func TestServiceUnblockTimeRange(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurfWithSlotSettings(t, svc, store)

	blocked, err := svc.BlockTimeRange(context.Background(), testUserID, turf.ID, BlockTimeRangeRequest{
		Date: "2026-09-01", StartTime: "10:00", EndTime: "12:00",
	})
	if err != nil {
		t.Fatalf("BlockTimeRange() returned error: %v", err)
	}
	if err := svc.UnblockTimeRange(context.Background(), testUserID, turf.ID, blocked.ID); err != nil {
		t.Fatalf("UnblockTimeRange() returned error: %v", err)
	}

	ranges, err := svc.BlockedTimeRanges(context.Background(), testUserID, turf.ID)
	if err != nil {
		t.Fatalf("BlockedTimeRanges() returned error: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("BlockedTimeRanges() = %d, want 0 after unblocking", len(ranges))
	}
}

func TestGenerateSlotsRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     GenerateSlotsRequest
		wantErr bool
	}{
		{"valid same day", GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-01"}, false},
		{"valid range", GenerateSlotsRequest{From: "2026-09-01", To: "2026-09-30"}, false},
		{"to before from", GenerateSlotsRequest{From: "2026-09-05", To: "2026-09-01"}, true},
		{"malformed from", GenerateSlotsRequest{From: "not-a-date", To: "2026-09-01"}, true},
		{"malformed to", GenerateSlotsRequest{From: "2026-09-01", To: "not-a-date"}, true},
		{"spans more than 31 days", GenerateSlotsRequest{From: "2026-01-01", To: "2026-03-01"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := len(tt.req.Validate()) > 0
			if got != tt.wantErr {
				t.Errorf("Validate() has errors = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestSlotSettingsRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     SlotSettingsRequest
		wantErr bool
	}{
		{"valid", SlotSettingsRequest{SlotDurationMinutes: 60, SlotPrice: 500}, false},
		{"zero price allowed", SlotSettingsRequest{SlotDurationMinutes: 30, SlotPrice: 0}, false},
		{"too short", SlotSettingsRequest{SlotDurationMinutes: 10, SlotPrice: 500}, true},
		{"too long", SlotSettingsRequest{SlotDurationMinutes: 300, SlotPrice: 500}, true},
		{"not a multiple of 15", SlotSettingsRequest{SlotDurationMinutes: 50, SlotPrice: 500}, true},
		{"negative price", SlotSettingsRequest{SlotDurationMinutes: 60, SlotPrice: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := len(tt.req.Validate()) > 0
			if got != tt.wantErr {
				t.Errorf("Validate() has errors = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestBlockTimeRangeRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     BlockTimeRangeRequest
		wantErr bool
	}{
		{"valid", BlockTimeRangeRequest{Date: "2026-09-01", StartTime: "10:00", EndTime: "12:00"}, false},
		{"end before start", BlockTimeRangeRequest{Date: "2026-09-01", StartTime: "12:00", EndTime: "10:00"}, true},
		{"end equals start", BlockTimeRangeRequest{Date: "2026-09-01", StartTime: "10:00", EndTime: "10:00"}, true},
		{"bad start format", BlockTimeRangeRequest{Date: "2026-09-01", StartTime: "10", EndTime: "12:00"}, true},
		{"bad date", BlockTimeRangeRequest{Date: "not-a-date", StartTime: "10:00", EndTime: "12:00"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := len(tt.req.Validate()) > 0
			if got != tt.wantErr {
				t.Errorf("Validate() has errors = %v, want %v", got, tt.wantErr)
			}
		})
	}
}
