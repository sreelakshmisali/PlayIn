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
	mu            sync.Mutex
	profiles      map[string]*profileRow // keyed by user id
	turfs         map[string]*turfRow    // keyed by turf id
	sports        map[string]SportRef
	amenities     map[string]Amenity
	turfSports    map[string]map[string]bool // turf id -> sport id -> present
	turfAmenities map[string]map[string]bool // turf id -> amenity id -> present
	turfImages    map[string][]TurfImage     // turf id -> images
	seq           int
	failWith      error
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
		turfSports:    make(map[string]map[string]bool),
		turfAmenities: make(map[string]map[string]bool),
		turfImages:    make(map[string][]TurfImage),
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
