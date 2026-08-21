package owners

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Seeded sport and amenity ids used by the in-memory store. They stand in for
// the UUIDs migrations 000003 and 000004 generate.
const (
	footballID     = "sport-football"
	cricketID      = "sport-cricket"
	retiredSportID = "sport-retired"

	parkingID        = "amenity-parking"
	floodlightsID    = "amenity-floodlights"
	retiredAmenityID = "amenity-retired"
)

// memStore is an in-memory Store. The repository's own SQL is exercised
// against a real database in repository_test.go and turf_repository_test.go;
// these tests are about the rules the service applies on top of it.
type memStore struct {
	mu                sync.Mutex
	profiles          map[string]*profileRow // keyed by user id
	turfs             map[string]*turfRow    // keyed by turf id
	sports            map[string]SportRef
	amenities         map[string]Amenity
	turfSports        map[string]map[string]bool // turf id -> sport id -> present
	turfAmenities     map[string]map[string]bool // turf id -> amenity id -> present
	turfImages        map[string][]TurfImage     // turf id -> images
	slots             map[string]*fakeSlot       // keyed by slot id
	blockedDates      map[string]*fakeBlockedDate
	blockedTimeRanges map[string]*fakeBlockedTimeRange
	seq               int
	failWith          error
}

// fakeSlot and the two block fakes below carry a TurfID that the real
// slotRow, BlockedDate and BlockedTimeRange never need to (see slot_model.go)
// because the fake, unlike the repository, has no turf-scoped SQL WHERE
// clause to do that filtering for it.
type fakeSlot struct {
	ID        string
	TurfID    string
	Date      string
	StartTime string
	EndTime   string
	Price     float64
	Status    SlotStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type fakeBlockedDate struct {
	ID     string
	TurfID string
	Date   string
	Reason string
}

type fakeBlockedTimeRange struct {
	ID        string
	TurfID    string
	Date      string
	StartTime string
	EndTime   string
	Reason    string
}

func newMemStore() *memStore {
	return &memStore{
		profiles: make(map[string]*profileRow),
		turfs:    make(map[string]*turfRow),
		sports: map[string]SportRef{
			footballID: {ID: footballID, Slug: "football", Name: "Football"},
			cricketID:  {ID: cricketID, Slug: "cricket", Name: "Cricket"},
			// retiredSportID intentionally absent: an inactive sport is not
			// selectable, so the fixture does not carry it as a valid choice.
		},
		amenities: map[string]Amenity{
			parkingID:     {ID: parkingID, Slug: "parking", Name: "Parking"},
			floodlightsID: {ID: floodlightsID, Slug: "floodlights", Name: "Floodlights"},
		},
		turfSports:        make(map[string]map[string]bool),
		turfAmenities:     make(map[string]map[string]bool),
		turfImages:        make(map[string][]TurfImage),
		slots:             make(map[string]*fakeSlot),
		blockedDates:      make(map[string]*fakeBlockedDate),
		blockedTimeRanges: make(map[string]*fakeBlockedTimeRange),
	}
}

func (m *memStore) nextID(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s-%d", prefix, m.seq)
}

// --- owner profile ---------------------------------------------------------

func (m *memStore) ProfileByUserID(_ context.Context, userID string) (Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Profile{}, m.failWith
	}
	row, ok := m.profiles[userID]
	if !ok {
		return Profile{}, ErrOwnerProfileNotFound
	}
	return row.toProfile(), nil
}

func (m *memStore) ProfileIDForUser(_ context.Context, userID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return "", m.failWith
	}
	row, ok := m.profiles[userID]
	if !ok {
		return "", ErrOwnerProfileNotFound
	}
	return row.ID, nil
}

func (m *memStore) SaveProfile(_ context.Context, userID string, f profileFields) (Profile, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Profile{}, false, m.failWith
	}

	now := time.Now().UTC()
	row, existed := m.profiles[userID]
	if !existed {
		row = &profileRow{ID: m.nextID("profile"), UserID: userID, CreatedAt: now}
		m.profiles[userID] = row
	}
	row.DisplayName = f.DisplayName
	row.Phone = f.Phone
	row.Description = f.Description
	row.UpdatedAt = now

	return row.toProfile(), !existed, nil
}

// --- turfs -------------------------------------------------------------------

func (m *memStore) CreateTurf(_ context.Context, ownerProfileID string, f turfFields) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}

	for _, t := range m.turfs {
		if t.OwnerID == ownerProfileID && strings.EqualFold(t.Name, f.Name) {
			return Turf{}, ErrTurfNameTaken
		}
	}

	now := time.Now().UTC()
	row := &turfRow{
		ID: m.nextID("turf"), OwnerID: ownerProfileID, OwnerDisplayName: m.displayNameFor(ownerProfileID),
		Name: f.Name, Description: f.Description, Address: f.Address, City: f.City,
		Latitude: f.Latitude, Longitude: f.Longitude, Capacity: f.Capacity,
		OpeningTime: f.OpeningTime, ClosingTime: f.ClosingTime,
		Status: StatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	m.turfs[row.ID] = row

	return m.assemble(row), nil
}

func (m *memStore) displayNameFor(ownerProfileID string) string {
	for _, p := range m.profiles {
		if p.ID == ownerProfileID {
			return p.DisplayName
		}
	}
	return ""
}

func (m *memStore) TurfsByOwner(_ context.Context, ownerProfileID string) ([]Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}

	var out []Turf
	for _, row := range m.turfs {
		if row.OwnerID == ownerProfileID {
			out = append(out, m.assemble(row))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *memStore) TurfByOwnerAndID(_ context.Context, ownerProfileID, turfID string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return Turf{}, ErrTurfNotFound
	}
	return m.assemble(row), nil
}

func (m *memStore) UpdateTurf(_ context.Context, ownerProfileID, turfID string, f turfFields) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return Turf{}, ErrTurfNotFound
	}
	for _, t := range m.turfs {
		if t.ID != turfID && t.OwnerID == ownerProfileID && strings.EqualFold(t.Name, f.Name) {
			return Turf{}, ErrTurfNameTaken
		}
	}

	row.Name, row.Description, row.Address, row.City = f.Name, f.Description, f.Address, f.City
	row.Latitude, row.Longitude, row.Capacity = f.Latitude, f.Longitude, f.Capacity
	row.OpeningTime, row.ClosingTime = f.OpeningTime, f.ClosingTime
	if row.Status == StatusApproved {
		row.Status = StatusPendingApproval
	}
	row.UpdatedAt = time.Now().UTC()

	return m.assemble(row), nil
}

func (m *memStore) DeleteTurf(_ context.Context, ownerProfileID, turfID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrTurfNotFound
	}
	delete(m.turfs, turfID)
	delete(m.turfSports, turfID)
	delete(m.turfAmenities, turfID)
	delete(m.turfImages, turfID)
	return nil
}

func (m *memStore) SubmitTurf(_ context.Context, ownerProfileID, turfID string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return Turf{}, ErrTurfNotFound
	}
	if row.Status != StatusDraft && row.Status != StatusRejected {
		return Turf{}, ErrInvalidStatusTransition
	}
	row.Status = StatusPendingApproval
	row.ModerationReason = ""
	row.UpdatedAt = time.Now().UTC()
	return m.assemble(row), nil
}

func (m *memStore) SetTurfSport(_ context.Context, ownerProfileID, turfID, sportID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrTurfNotFound
	}
	if _, ok := m.sports[sportID]; !ok {
		return ErrSportNotFound
	}
	if m.turfSports[turfID] == nil {
		m.turfSports[turfID] = make(map[string]bool)
	}
	m.turfSports[turfID][sportID] = true
	return nil
}

func (m *memStore) RemoveTurfSport(_ context.Context, ownerProfileID, turfID, sportID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID || !m.turfSports[turfID][sportID] {
		return ErrTurfSportNotFound
	}
	delete(m.turfSports[turfID], sportID)
	return nil
}

func (m *memStore) SetTurfAmenity(_ context.Context, ownerProfileID, turfID, amenityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrTurfNotFound
	}
	if _, ok := m.amenities[amenityID]; !ok {
		return ErrAmenityNotFound
	}
	if m.turfAmenities[turfID] == nil {
		m.turfAmenities[turfID] = make(map[string]bool)
	}
	m.turfAmenities[turfID][amenityID] = true
	return nil
}

func (m *memStore) RemoveTurfAmenity(_ context.Context, ownerProfileID, turfID, amenityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID || !m.turfAmenities[turfID][amenityID] {
		return ErrTurfAmenityNotFound
	}
	delete(m.turfAmenities[turfID], amenityID)
	return nil
}

func (m *memStore) AddTurfImage(_ context.Context, ownerProfileID, turfID, imageURL string) (TurfImage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return TurfImage{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return TurfImage{}, ErrTurfNotFound
	}
	if len(m.turfImages[turfID]) >= maxTurfImages {
		return TurfImage{}, ErrTooManyImages
	}
	img := TurfImage{ID: m.nextID("image"), ImageURL: imageURL}
	m.turfImages[turfID] = append(m.turfImages[turfID], img)
	return img, nil
}

func (m *memStore) RemoveTurfImage(_ context.Context, ownerProfileID, turfID, imageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrTurfImageNotFound
	}
	images := m.turfImages[turfID]
	for i, img := range images {
		if img.ID == imageID {
			m.turfImages[turfID] = append(images[:i], images[i+1:]...)
			return nil
		}
	}
	return ErrTurfImageNotFound
}

func (m *memStore) Amenities(context.Context) ([]Amenity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}
	out := make([]Amenity, 0, len(m.amenities))
	for _, a := range m.amenities {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *memStore) PublicTurfs(context.Context) ([]Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}
	var out []Turf
	for _, row := range m.turfs {
		if row.Status == StatusApproved {
			out = append(out, m.assemble(row))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *memStore) PublicTurfByID(_ context.Context, turfID string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.Status != StatusApproved {
		return Turf{}, ErrTurfNotFound
	}
	return m.assemble(row), nil
}

// --- admin moderation --------------------------------------------------------

func (m *memStore) PendingTurfs(context.Context) ([]Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}
	var out []Turf
	for _, row := range m.turfs {
		if row.Status == StatusPendingApproval {
			out = append(out, m.assemble(row))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (m *memStore) TurfByID(_ context.Context, turfID string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok {
		return Turf{}, ErrTurfNotFound
	}
	return m.assemble(row), nil
}

func (m *memStore) ApproveTurf(_ context.Context, turfID, _ string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok {
		return Turf{}, ErrTurfNotFound
	}
	if row.Status != StatusPendingApproval {
		return Turf{}, ErrInvalidStatusTransition
	}
	row.Status = StatusApproved
	row.ModerationReason = ""
	row.UpdatedAt = time.Now().UTC()
	return m.assemble(row), nil
}

func (m *memStore) RejectTurf(_ context.Context, turfID, _, reason string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok {
		return Turf{}, ErrTurfNotFound
	}
	if row.Status != StatusPendingApproval {
		return Turf{}, ErrInvalidStatusTransition
	}
	row.Status = StatusRejected
	row.ModerationReason = reason
	row.UpdatedAt = time.Now().UTC()
	return m.assemble(row), nil
}

func (m *memStore) SuspendTurf(_ context.Context, turfID, _, reason string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok {
		return Turf{}, ErrTurfNotFound
	}
	if row.Status != StatusApproved {
		return Turf{}, ErrInvalidStatusTransition
	}
	row.Status = StatusSuspended
	row.ModerationReason = reason
	row.UpdatedAt = time.Now().UTC()
	return m.assemble(row), nil
}

func (m *memStore) RestoreTurf(_ context.Context, turfID, _ string) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok {
		return Turf{}, ErrTurfNotFound
	}
	if row.Status != StatusSuspended {
		return Turf{}, ErrInvalidStatusTransition
	}
	row.Status = StatusApproved
	row.ModerationReason = ""
	row.UpdatedAt = time.Now().UTC()
	return m.assemble(row), nil
}

// --- slots and availability --------------------------------------------------

func (m *memStore) UpdateSlotSettings(_ context.Context, ownerProfileID, turfID string, f slotSettingsFields) (Turf, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Turf{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return Turf{}, ErrTurfNotFound
	}

	duration, price := f.DurationMinutes, f.Price
	row.SlotDurationMinutes = &duration
	row.SlotPrice = &price
	row.UpdatedAt = time.Now().UTC()
	return m.assemble(row), nil
}

func (m *memStore) SlotsInRange(_ context.Context, ownerProfileID, turfID string, from, to time.Time) ([]Slot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return nil, ErrTurfNotFound
	}

	var out []Slot
	for _, s := range m.slots {
		if s.TurfID != turfID {
			continue
		}
		d, err := time.Parse(dateLayout, s.Date)
		if err != nil || d.Before(from) || d.After(to) {
			continue
		}
		out = append(out, m.toSlot(s))
	}
	sortSlots(out)
	return out, nil
}

func (m *memStore) PublicSlotsForDate(_ context.Context, turfID string, date time.Time) ([]Slot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.Status != StatusApproved {
		return nil, ErrTurfNotFound
	}

	dateStr := date.Format(dateLayout)
	var out []Slot
	for _, s := range m.slots {
		if s.TurfID == turfID && s.Date == dateStr {
			out = append(out, m.toSlot(s))
		}
	}
	sortSlots(out)
	return out, nil
}

func sortSlots(slots []Slot) {
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].Date != slots[j].Date {
			return slots[i].Date < slots[j].Date
		}
		return slots[i].StartTime < slots[j].StartTime
	})
}

// InsertSlots mirrors the repository's ON CONFLICT DO NOTHING: a candidate
// that overlaps a slot already on this turf and date is silently skipped
// rather than erroring, the same idempotent-retry behaviour generation gets
// from the real turf_slots_no_overlap exclusion constraint.
func (m *memStore) InsertSlots(_ context.Context, ownerProfileID, turfID string, candidates []slotCandidate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrTurfNotFound
	}

	for _, c := range candidates {
		dateStr := c.Date.Format(dateLayout)
		if m.slotOverlapsExisting(turfID, dateStr, c.StartTime, c.EndTime) {
			continue
		}
		now := time.Now().UTC()
		id := m.nextID("slot")
		m.slots[id] = &fakeSlot{
			ID: id, TurfID: turfID, Date: dateStr,
			StartTime: c.StartTime, EndTime: c.EndTime, Price: c.Price,
			Status: SlotStatusOpen, CreatedAt: now, UpdatedAt: now,
		}
	}
	return nil
}

func (m *memStore) slotOverlapsExisting(turfID, date, start, end string) bool {
	for _, s := range m.slots {
		if s.TurfID == turfID && s.Date == date && rangesOverlap(start, end, s.StartTime, s.EndTime) {
			return true
		}
	}
	return false
}

// rangesOverlap reports whether two half-open HH:MM intervals intersect.
// String comparison is valid ordering for zero-padded 24-hour HH:MM values.
func rangesOverlap(aStart, aEnd, bStart, bEnd string) bool {
	return aStart < bEnd && bStart < aEnd
}

func (m *memStore) SetSlotStatus(_ context.Context, ownerProfileID, turfID, slotID string, status SlotStatus) (Slot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Slot{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return Slot{}, ErrSlotNotFound
	}
	s, ok := m.slots[slotID]
	if !ok || s.TurfID != turfID {
		return Slot{}, ErrSlotNotFound
	}

	s.Status = status
	s.UpdatedAt = time.Now().UTC()
	return m.toSlot(s), nil
}

func (m *memStore) DeleteSlot(_ context.Context, ownerProfileID, turfID, slotID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrSlotNotFound
	}
	s, ok := m.slots[slotID]
	if !ok || s.TurfID != turfID {
		return ErrSlotNotFound
	}
	delete(m.slots, slotID)
	return nil
}

// toSlot computes availability the same way the repository's slotColumns
// projection does: OPEN and outside both kinds of block. The caller holds
// the lock.
func (m *memStore) toSlot(s *fakeSlot) Slot {
	available := s.Status == SlotStatusOpen
	if available {
		for _, b := range m.blockedDates {
			if b.TurfID == s.TurfID && b.Date == s.Date {
				available = false
				break
			}
		}
	}
	if available {
		for _, b := range m.blockedTimeRanges {
			if b.TurfID == s.TurfID && b.Date == s.Date && rangesOverlap(s.StartTime, s.EndTime, b.StartTime, b.EndTime) {
				available = false
				break
			}
		}
	}
	return Slot{
		ID: s.ID, Date: s.Date, StartTime: s.StartTime, EndTime: s.EndTime,
		Price: s.Price, Status: s.Status, Available: available,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

// --- blocked dates -------------------------------------------------------------

func (m *memStore) BlockedDates(_ context.Context, ownerProfileID, turfID string) ([]BlockedDate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return nil, ErrTurfNotFound
	}

	var out []BlockedDate
	for _, b := range m.blockedDates {
		if b.TurfID == turfID {
			out = append(out, BlockedDate{ID: b.ID, Date: b.Date, Reason: b.Reason})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out, nil
}

func (m *memStore) BlockDate(_ context.Context, ownerProfileID, turfID string, date time.Time, reason string) (BlockedDate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return BlockedDate{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return BlockedDate{}, ErrTurfNotFound
	}

	dateStr := date.Format(dateLayout)
	for _, b := range m.blockedDates {
		if b.TurfID == turfID && b.Date == dateStr {
			return BlockedDate{}, ErrDateAlreadyBlocked
		}
	}

	id := m.nextID("blocked-date")
	m.blockedDates[id] = &fakeBlockedDate{ID: id, TurfID: turfID, Date: dateStr, Reason: reason}
	return BlockedDate{ID: id, Date: dateStr, Reason: reason}, nil
}

func (m *memStore) UnblockDate(_ context.Context, ownerProfileID, turfID, blockedDateID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrBlockedDateNotFound
	}
	b, ok := m.blockedDates[blockedDateID]
	if !ok || b.TurfID != turfID {
		return ErrBlockedDateNotFound
	}
	delete(m.blockedDates, blockedDateID)
	return nil
}

// --- blocked time ranges -------------------------------------------------------

func (m *memStore) BlockedTimeRanges(_ context.Context, ownerProfileID, turfID string) ([]BlockedTimeRange, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return nil, ErrTurfNotFound
	}

	var out []BlockedTimeRange
	for _, b := range m.blockedTimeRanges {
		if b.TurfID == turfID {
			out = append(out, BlockedTimeRange{
				ID: b.ID, Date: b.Date, StartTime: b.StartTime, EndTime: b.EndTime, Reason: b.Reason,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].StartTime < out[j].StartTime
	})
	return out, nil
}

func (m *memStore) BlockTimeRange(_ context.Context, ownerProfileID, turfID string, date time.Time, startTime, endTime, reason string) (BlockedTimeRange, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return BlockedTimeRange{}, m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return BlockedTimeRange{}, ErrTurfNotFound
	}

	dateStr := date.Format(dateLayout)
	for _, b := range m.blockedTimeRanges {
		if b.TurfID == turfID && b.Date == dateStr && rangesOverlap(startTime, endTime, b.StartTime, b.EndTime) {
			return BlockedTimeRange{}, ErrTimeRangeOverlapsBlock
		}
	}

	id := m.nextID("blocked-range")
	m.blockedTimeRanges[id] = &fakeBlockedTimeRange{
		ID: id, TurfID: turfID, Date: dateStr, StartTime: startTime, EndTime: endTime, Reason: reason,
	}
	return BlockedTimeRange{ID: id, Date: dateStr, StartTime: startTime, EndTime: endTime, Reason: reason}, nil
}

func (m *memStore) UnblockTimeRange(_ context.Context, ownerProfileID, turfID, blockedTimeRangeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	row, ok := m.turfs[turfID]
	if !ok || row.OwnerID != ownerProfileID {
		return ErrBlockedTimeRangeNotFound
	}
	b, ok := m.blockedTimeRanges[blockedTimeRangeID]
	if !ok || b.TurfID != turfID {
		return ErrBlockedTimeRangeNotFound
	}
	delete(m.blockedTimeRanges, blockedTimeRangeID)
	return nil
}

// assemble builds the response projection, mirroring how the repository joins
// sports, amenities and images onto a turf row. The caller holds the lock.
func (m *memStore) assemble(row *turfRow) Turf {
	var sports []SportRef
	for id := range m.turfSports[row.ID] {
		sports = append(sports, m.sports[id])
	}
	sort.Slice(sports, func(i, j int) bool { return sports[i].Name < sports[j].Name })

	var amenities []Amenity
	for id := range m.turfAmenities[row.ID] {
		amenities = append(amenities, m.amenities[id])
	}
	sort.Slice(amenities, func(i, j int) bool { return amenities[i].Name < amenities[j].Name })

	return row.toTurf(sports, amenities, m.turfImages[row.ID])
}

// setStatus is a test helper that reaches past the service to force a status
// directly, bypassing transition rules. Tests that need to set up a turf in a
// given status without exercising the transition that would normally produce
// it use this; tests of the transitions themselves call the admin methods.
func (m *memStore) setStatus(turfID string, status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if row, ok := m.turfs[turfID]; ok {
		row.Status = status
	}
}

func newTestService() (*Service, *memStore) {
	store := newMemStore()
	return NewService(store), store
}

// stubAuthenticator resolves any bearer token to a fixed user, so the handler
// tests exercise the real guards without minting real JWTs.
type stubAuthenticator struct {
	user auth.User
	err  error
}

func (s stubAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return s.user, s.err
}

func testUser(role auth.Role) auth.User {
	return auth.User{ID: "user-1", Email: "owner@playhub.test", Role: role, IsActive: true}
}

func newTestHandler(user auth.User) (*Handler, *memStore) {
	svc, store := newTestService()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(svc, stubAuthenticator{user: user}, logger), store
}

// validSaveProfileRequest is the happy-path owner profile body.
func validSaveProfileRequest() SaveProfileRequest {
	return SaveProfileRequest{
		DisplayName: "Kochi Sports Arena",
		Phone:       "+91 98765 43210",
		Description: "Multi-sport turf in the heart of the city.",
	}
}

// seedProfile saves an owner profile through the service.
func seedProfile(t *testing.T, svc *Service, userID string) Profile {
	t.Helper()

	req := validSaveProfileRequest()
	req.Normalise()

	profile, _, err := svc.SaveProfile(context.Background(), userID, req)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	return profile
}

// validSaveTurfRequest is the happy-path turf body.
func validSaveTurfRequest() SaveTurfRequest {
	lat, lng := 9.9312, 76.2673
	capacity := int32(22)
	return SaveTurfRequest{
		Name:        "Riverside Turf",
		Description: "A well-kept five-a-side turf by the river.",
		Address:     "123 River Road, Panampilly Nagar",
		City:        "Kochi",
		Latitude:    &lat,
		Longitude:   &lng,
		Capacity:    &capacity,
		OpeningTime: "06:00",
		ClosingTime: "22:00",
	}
}

// seedTurf creates a turf for userID through the service, seeding their owner
// profile first if it does not already exist.
func seedTurf(t *testing.T, svc *Service, store *memStore, userID string) Turf {
	t.Helper()

	if _, err := store.ProfileIDForUser(context.Background(), userID); err != nil {
		seedProfile(t, svc, userID)
	}

	req := validSaveTurfRequest()
	req.Normalise()

	turf, err := svc.CreateTurf(context.Background(), userID, req)
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	return turf
}

// fieldNames lists the field names in a validation result, for comparison.
func fieldNames(errs []httpx.FieldError) string {
	names := make([]string, 0, len(errs))
	for _, e := range errs {
		names = append(names, e.Field)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
