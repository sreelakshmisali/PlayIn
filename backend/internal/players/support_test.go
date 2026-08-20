package players

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

// Seeded sport ids used by the in-memory store. They stand in for the UUIDs
// migration 000003 generates.
const (
	footballID  = "sport-football"
	cricketID   = "sport-cricket"
	badmintonID = "sport-badminton"
	retiredID   = "sport-retired"
)

func testSports() []Sport {
	return []Sport{
		{ID: badmintonID, Slug: "badminton", Name: "Badminton", Positions: []string{}},
		{ID: cricketID, Slug: "cricket", Name: "Cricket", Positions: []string{"Batter", "Bowler", "All-rounder", "Wicketkeeper"}},
		{ID: footballID, Slug: "football", Name: "Football", Positions: []string{"Goalkeeper", "Defender", "Midfielder", "Forward"}},
	}
}

// memStore is an in-memory Store. The repository's own SQL is exercised against
// a real database in repository_test.go; these tests are about the rules the
// service applies on top of it.
type memStore struct {
	mu       sync.Mutex
	sports   map[string]Sport
	profiles map[string]*profileRow       // keyed by user id
	chosen   map[string]map[string]string // profile id -> sport id -> position
	seq      int
	failWith error
}

func newMemStore() *memStore {
	m := &memStore{
		sports:   make(map[string]Sport),
		profiles: make(map[string]*profileRow),
		chosen:   make(map[string]map[string]string),
	}
	for _, s := range testSports() {
		m.sports[s.ID] = s
	}
	// A retired sport, to prove it cannot be chosen or listed.
	m.sports[retiredID] = Sport{ID: retiredID, Slug: "kabaddi", Name: "Kabaddi", Positions: []string{"Raider"}}
	return m
}

// active reports whether a sport is selectable. Only the retired fixture is not.
func active(id string) bool { return id != retiredID }

func (m *memStore) Sports(context.Context) ([]Sport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}

	out := make([]Sport, 0, len(m.sports))
	for id, s := range m.sports {
		if active(id) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *memStore) SportByID(_ context.Context, id string) (Sport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Sport{}, m.failWith
	}
	s, ok := m.sports[id]
	if !ok || !active(id) {
		return Sport{}, ErrSportNotFound
	}
	return s, nil
}

func (m *memStore) ProfileByUserID(_ context.Context, userID string) (Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return Profile{}, m.failWith
	}
	row, ok := m.profiles[userID]
	if !ok {
		return Profile{}, ErrProfileNotFound
	}
	return row.toProfile(m.sportsFor(row.ID)), nil
}

func (m *memStore) ProfileIDForUser(_ context.Context, userID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return "", m.failWith
	}
	row, ok := m.profiles[userID]
	if !ok {
		return "", ErrProfileNotFound
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
		m.seq++
		row = &profileRow{ID: fmt.Sprintf("profile-%d", m.seq), UserID: userID, CreatedAt: now}
		m.profiles[userID] = row
	}

	row.DisplayName = f.DisplayName
	row.ImageURL = f.ImageURL
	row.Bio = f.Bio
	row.Location = f.Location
	row.UpdatedAt = now

	return row.toProfile(m.sportsFor(row.ID)), !existed, nil
}

func (m *memStore) SetSport(_ context.Context, profileID, sportID, position string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}

	// Mirrors the guarded INSERT: the sport must exist, be active, and offer
	// the position before a row is written.
	sport, ok := m.sports[sportID]
	if !ok || !active(sportID) {
		return ErrSportNotFound
	}
	if position != "" && !sport.HasPosition(position) {
		return ErrSportNotFound
	}

	if m.chosen[profileID] == nil {
		m.chosen[profileID] = make(map[string]string)
	}
	m.chosen[profileID][sportID] = position
	return nil
}

func (m *memStore) RemoveSport(_ context.Context, profileID, sportID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	if _, ok := m.chosen[profileID][sportID]; !ok {
		return ErrSportNotPreferred
	}
	delete(m.chosen[profileID], sportID)
	return nil
}

// sportsFor builds the joined view. The caller holds the lock.
func (m *memStore) sportsFor(profileID string) []PlayerSport {
	picked := m.chosen[profileID]
	out := make([]PlayerSport, 0, len(picked))

	for sportID, position := range picked {
		out = append(out, PlayerSport{Sport: m.sports[sportID], Position: position})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sport.Name < out[j].Sport.Name })
	return out
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
	return auth.User{ID: "user-1", Email: "player@playhub.test", Role: role, IsActive: true}
}

func newTestHandler(user auth.User) (*Handler, *memStore) {
	svc, store := newTestService()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewHandler(svc, stubAuthenticator{user: user}, logger), store
}

// validSaveRequest is the happy-path profile body.
func validSaveRequest() SaveProfileRequest {
	return SaveProfileRequest{
		DisplayName: "Priya Raman",
		ImageURL:    "https://cdn.playhub.test/priya.jpg",
		Bio:         "Weekend midfielder.",
		Location:    "Kochi",
	}
}

// seedProfile saves a profile through the service.
func seedProfile(t *testing.T, svc *Service, userID string) Profile {
	t.Helper()

	req := validSaveRequest()
	req.Normalise()

	profile, _, err := svc.SaveProfile(context.Background(), userID, req)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	return profile
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
