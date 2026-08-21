package owners

import (
	"context"
	"fmt"
	"time"
)

// UpdateSlotSettings sets the slot duration and price one of the caller's own
// turfs generates slots with.
func (s *Service) UpdateSlotSettings(ctx context.Context, userID, turfID string, req SlotSettingsRequest) (Turf, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Turf{}, err
	}
	return s.store.UpdateSlotSettings(ctx, ownerProfileID, turfID, req.fields())
}

// GenerateSlots creates every slot the turf's operating hours, slot duration
// and blocks allow across a date range, and returns the full slot list for
// that range, newly created and pre-existing alike.
//
// Generation is idempotent: a candidate that already exists is silently
// skipped (see Repository.InsertSlots), so calling this again over an
// overlapping range, or after changing a block, is always safe to retry.
func (s *Service) GenerateSlots(ctx context.Context, userID, turfID string, req GenerateSlotsRequest) ([]Slot, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	turf, err := s.store.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
	if err != nil {
		return nil, err
	}
	if turf.SlotDurationMinutes == nil || turf.SlotPrice == nil {
		return nil, ErrSlotSettingsNotConfigured
	}

	from, to, err := parseRange(req.From, req.To)
	if err != nil {
		return nil, err
	}

	blockedDates, err := s.store.BlockedDates(ctx, ownerProfileID, turfID)
	if err != nil {
		return nil, err
	}
	blockedRanges, err := s.store.BlockedTimeRanges(ctx, ownerProfileID, turfID)
	if err != nil {
		return nil, err
	}

	candidates := buildSlotCandidates(turf, from, to, blockedDates, blockedRanges)
	if err := s.store.InsertSlots(ctx, ownerProfileID, turfID, candidates); err != nil {
		return nil, err
	}

	return s.store.SlotsInRange(ctx, ownerProfileID, turfID, from, to)
}

// SlotsInRange lists one of the caller's own turf's slots over a date range,
// generated or not: the owner's availability view.
func (s *Service) SlotsInRange(ctx context.Context, userID, turfID string, req SlotRangeQuery) ([]Slot, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	from, to, err := parseRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	return s.store.SlotsInRange(ctx, ownerProfileID, turfID, from, to)
}

// PublicSlotsForDate lists an APPROVED turf's slots for one date, for anyone
// to browse.
func (s *Service) PublicSlotsForDate(ctx context.Context, turfID, date string) ([]Slot, error) {
	d, err := parseDate(date)
	if err != nil {
		return nil, ErrInvalidDateRange
	}
	return s.store.PublicSlotsForDate(ctx, turfID, d)
}

// SetSlotStatus toggles one of the caller's own slots between OPEN and
// BLOCKED.
func (s *Service) SetSlotStatus(ctx context.Context, userID, turfID, slotID string, req SetSlotStatusRequest) (Slot, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return Slot{}, err
	}
	return s.store.SetSlotStatus(ctx, ownerProfileID, turfID, slotID, req.Status)
}

// DeleteSlot removes one of the caller's own generated slots.
func (s *Service) DeleteSlot(ctx context.Context, userID, turfID, slotID string) error {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.store.DeleteSlot(ctx, ownerProfileID, turfID, slotID)
}

// BlockedDates lists every blocked date on one of the caller's own turfs.
func (s *Service) BlockedDates(ctx context.Context, userID, turfID string) ([]BlockedDate, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.store.BlockedDates(ctx, ownerProfileID, turfID)
}

// BlockDate blocks a whole day on one of the caller's own turfs.
func (s *Service) BlockDate(ctx context.Context, userID, turfID string, req BlockDateRequest) (BlockedDate, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return BlockedDate{}, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return BlockedDate{}, ErrInvalidDateRange
	}
	return s.store.BlockDate(ctx, ownerProfileID, turfID, date, req.Reason)
}

// UnblockDate removes a blocked date from one of the caller's own turfs.
func (s *Service) UnblockDate(ctx context.Context, userID, turfID, blockedDateID string) error {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.store.UnblockDate(ctx, ownerProfileID, turfID, blockedDateID)
}

// BlockedTimeRanges lists every blocked time range on one of the caller's own
// turfs.
func (s *Service) BlockedTimeRanges(ctx context.Context, userID, turfID string) ([]BlockedTimeRange, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.store.BlockedTimeRanges(ctx, ownerProfileID, turfID)
}

// BlockTimeRange blocks part of one day on one of the caller's own turfs.
func (s *Service) BlockTimeRange(ctx context.Context, userID, turfID string, req BlockTimeRangeRequest) (BlockedTimeRange, error) {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return BlockedTimeRange{}, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return BlockedTimeRange{}, ErrInvalidDateRange
	}
	return s.store.BlockTimeRange(ctx, ownerProfileID, turfID, date, req.StartTime, req.EndTime, req.Reason)
}

// UnblockTimeRange removes a blocked time range from one of the caller's own
// turfs.
func (s *Service) UnblockTimeRange(ctx context.Context, userID, turfID, blockedTimeRangeID string) error {
	ownerProfileID, err := s.store.ProfileIDForUser(ctx, userID)
	if err != nil {
		return err
	}
	return s.store.UnblockTimeRange(ctx, ownerProfileID, turfID, blockedTimeRangeID)
}

func parseRange(from, to string) (time.Time, time.Time, error) {
	fromDate, err := parseDate(from)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidDateRange
	}
	toDate, err := parseDate(to)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidDateRange
	}
	if toDate.Before(fromDate) {
		return time.Time{}, time.Time{}, ErrInvalidDateRange
	}
	return fromDate, toDate, nil
}

// buildSlotCandidates walks the turf's operating hours in slot-duration steps
// for every date in [from, to], skipping a date blocked outright and any
// candidate that would overlap a blocked time range. The grid is
// deterministic, so two candidates it produces never overlap each other; the
// exclusion constraints in migration 000006 are what guard against overlap
// with anything generated by an earlier call.
func buildSlotCandidates(turf Turf, from, to time.Time, blockedDates []BlockedDate, blockedRanges []BlockedTimeRange) []slotCandidate {
	blockedDateSet := make(map[string]bool, len(blockedDates))
	for _, bd := range blockedDates {
		blockedDateSet[bd.Date] = true
	}

	rangesByDate := make(map[string][]BlockedTimeRange, len(blockedRanges))
	for _, br := range blockedRanges {
		rangesByDate[br.Date] = append(rangesByDate[br.Date], br)
	}

	duration := int(*turf.SlotDurationMinutes)
	openMinute := minutesSinceMidnight(turf.OpeningTime)
	closeMinute := minutesSinceMidnight(turf.ClosingTime)
	price := *turf.SlotPrice

	var candidates []slotCandidate
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format(dateLayout)
		if blockedDateSet[dateStr] {
			continue
		}
		ranges := rangesByDate[dateStr]

		for start := openMinute; start+duration <= closeMinute; start += duration {
			end := start + duration
			if overlapsAnyRange(start, end, ranges) {
				continue
			}
			candidates = append(candidates, slotCandidate{
				Date:      d,
				StartTime: formatMinutes(start),
				EndTime:   formatMinutes(end),
				Price:     price,
			})
		}
	}
	return candidates
}

func overlapsAnyRange(start, end int, ranges []BlockedTimeRange) bool {
	for _, r := range ranges {
		rangeStart := minutesSinceMidnight(r.StartTime)
		rangeEnd := minutesSinceMidnight(r.EndTime)
		if start < rangeEnd && rangeStart < end {
			return true
		}
	}
	return false
}

// minutesSinceMidnight converts an HH:MM string to minutes since midnight.
// The caller is always a value that already passed timePattern validation
// (a turf's opening/closing time, or a blocked time range's own bound), so
// this indexes the string directly rather than parsing it generically.
func minutesSinceMidnight(hhmm string) int {
	h := int(hhmm[0]-'0')*10 + int(hhmm[1]-'0')
	m := int(hhmm[3]-'0')*10 + int(hhmm[4]-'0')
	return h*60 + m
}

func formatMinutes(totalMinutes int) string {
	return fmt.Sprintf("%02d:%02d", totalMinutes/60, totalMinutes%60)
}
