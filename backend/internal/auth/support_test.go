package auth

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

	"github.com/orgmelethil/playhub/backend/internal/config"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// memStore is an in-memory Store. The repository's own SQL is exercised by the
// end-to-end run against a real database; these tests are about the rules the
// service applies on top of it.
type memStore struct {
	mu       sync.Mutex
	users    map[string]User          // keyed by id
	tokens   map[string]RefreshRecord // keyed by id
	seq      int
	failWith error // when set, every method returns it
}

func newMemStore() *memStore {
	return &memStore{
		users:  make(map[string]User),
		tokens: make(map[string]RefreshRecord),
	}
}

func (m *memStore) nextID(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s-%d", prefix, m.seq)
}

func (m *memStore) CreateUser(_ context.Context, email, hash, fullName string, role Role) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return User{}, m.failWith
	}
	for _, u := range m.users {
		if u.Email == email {
			return User{}, ErrEmailTaken
		}
	}

	user := User{
		ID:           m.nextID("user"),
		Email:        email,
		PasswordHash: hash,
		FullName:     fullName,
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	m.users[user.ID] = user
	return user, nil
}

func (m *memStore) UserByEmail(_ context.Context, email string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return User{}, m.failWith
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (m *memStore) UserByID(_ context.Context, id string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return User{}, m.failWith
	}
	u, ok := m.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (m *memStore) CreateRefreshToken(_ context.Context, userID string, expiresAt time.Time) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return "", m.failWith
	}
	id := m.nextID("rt")
	m.tokens[id] = RefreshRecord{ID: id, UserID: userID, ExpiresAt: expiresAt}
	return id, nil
}

func (m *memStore) RefreshTokenByID(_ context.Context, id string) (RefreshRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return RefreshRecord{}, m.failWith
	}
	rec, ok := m.tokens[id]
	if !ok {
		return RefreshRecord{}, ErrInvalidToken
	}
	return rec, nil
}

func (m *memStore) RevokeRefreshToken(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return m.failWith
	}
	if rec, ok := m.tokens[id]; ok && rec.RevokedAt == nil {
		now := time.Now().UTC()
		rec.RevokedAt = &now
		m.tokens[id] = rec
	}
	return nil
}

// setActive flips a stored account's is_active flag.
func (m *memStore) setActive(id string, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u, ok := m.users[id]; ok {
		u.IsActive = active
		m.users[id] = u
	}
}

// Users lists accounts newest first, matching the repository's ORDER BY.
func (m *memStore) Users(_ context.Context, limit, offset int) ([]User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return nil, m.failWith
	}

	all := make([]User, 0, len(m.users))
	for _, u := range m.users {
		all = append(all, u)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })

	if offset >= len(all) {
		return []User{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *memStore) UserCount(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failWith != nil {
		return 0, m.failWith
	}
	return len(m.users), nil
}

// SetActive is the Store method the service calls; it wraps the test-only
// setActive helper so both the direct test shortcut and the real service path
// are exercised where each is needed.
func (m *memStore) SetActive(_ context.Context, id string, active bool) (User, error) {
	m.mu.Lock()
	if m.failWith != nil {
		defer m.mu.Unlock()
		return User{}, m.failWith
	}
	u, ok := m.users[id]
	if !ok {
		m.mu.Unlock()
		return User{}, ErrUserNotFound
	}
	u.IsActive = active
	m.users[id] = u
	m.mu.Unlock()
	return u, nil
}

// fakeHasher is a Hasher that does no work, so a test that exercises the login
// path does not pay bcrypt's cost per case.
type fakeHasher struct{}

const fakePrefix = "hashed:"

func (fakeHasher) Hash(password string) (string, error) { return fakePrefix + password, nil }

func (fakeHasher) Verify(hash, password string) error {
	if hash == fakePrefix+password {
		return nil
	}
	return ErrInvalidCredentials
}

func testAuthConfig() config.Auth {
	return config.Auth{
		JWTSecret:  "0123456789abcdef0123456789abcdef",
		JWTIssuer:  "playhub-test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
		BcryptCost: 4,
	}
}

// newTestService wires a service over the in-memory store.
func newTestService(t *testing.T) (*Service, *memStore) {
	t.Helper()

	store := newMemStore()
	return NewService(store, fakeHasher{}, NewIssuer(testAuthConfig())), store
}

func newTestHandler(t *testing.T) (*Handler, *memStore) {
	t.Helper()

	svc, store := newTestService(t)
	return NewHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil))), store
}

// validRegisterRequest is the happy-path signup body.
func validRegisterRequest() RegisterRequest {
	return RegisterRequest{
		Email:    "player@playhub.test",
		Password: "correct horse 7",
		FullName: "Test Player",
		Role:     RolePlayer,
	}
}

// registerUser signs a user up through the service and returns the session.
func registerUser(t *testing.T, svc *Service, email string, role Role) Session {
	t.Helper()

	req := validRegisterRequest()
	req.Email = email
	req.Role = role
	req.Normalise()

	session, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("Register(%q) returned error: %v", email, err)
	}
	return session
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
