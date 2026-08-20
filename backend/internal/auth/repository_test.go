package auth

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
// the schema: the unique index on email, the CHECK constraints, the cascade
// from users to refresh_tokens. A fake would only re-test the fake.
//
// They are skipped unless PLAYHUB_TEST_DATABASE_URL points at a migrated
// database, so `go test ./...` stays runnable without one:
//
//	PLAYHUB_TEST_DATABASE_URL=postgres://playhub:pass@localhost:5432/playhub?sslmode=disable go test ./internal/auth
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

// uniqueEmail keeps concurrent and repeated runs from colliding on the unique
// index, so the tests need no truncation between runs.
func uniqueEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%s-%d@playhub.test", sanitise(t.Name()), time.Now().UnixNano())
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

// createTestUser inserts an account and removes it when the test ends.
func createTestUser(t *testing.T, repo *Repository, role Role) User {
	t.Helper()

	ctx := context.Background()
	user, err := repo.CreateUser(ctx, uniqueEmail(t), "$2a$04$notarealhashbutlongenough", "Test Player", role)
	if err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}

	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID); err != nil {
			t.Errorf("cleaning up user %s failed: %v", user.ID, err)
		}
	})

	return user
}

func TestRepositoryCreateUser(t *testing.T) {
	repo := newTestRepository(t)

	user := createTestUser(t, repo, RolePlayer)

	if user.ID == "" {
		t.Error("CreateUser() returned an empty id, want a generated UUID")
	}
	if user.Role != RolePlayer {
		t.Errorf("Role = %q, want PLAYER", user.Role)
	}
	if !user.IsActive {
		t.Error("IsActive = false, want the column default of true")
	}
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Error("the timestamps are zero, want the column defaults")
	}
}

// The unique index is what actually settles a race between two simultaneous
// signups for the same address, so it is checked directly.
func TestRepositoryCreateUserRejectsDuplicateEmail(t *testing.T) {
	repo := newTestRepository(t)
	user := createTestUser(t, repo, RolePlayer)

	_, err := repo.CreateUser(context.Background(), user.Email, "$2a$04$notarealhashbutlongenough", "Someone Else", RoleOwner)
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("CreateUser() error = %v, want ErrEmailTaken", err)
	}
}

func TestRepositoryCreateUserRejectsBadInput(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		email    string
		hash     string
		fullName string
		role     Role
	}{
		{"uppercase email", "Player@Playhub.test", "$2a$04$notarealhashbutlongenough", "Test Player", RolePlayer},
		{"malformed email", "not-an-email", "$2a$04$notarealhashbutlongenough", "Test Player", RolePlayer},
		{"short hash", uniqueEmail(t), "plaintext", "Test Player", RolePlayer},
		{"blank name", uniqueEmail(t), "$2a$04$notarealhashbutlongenough", " ", RolePlayer},
		{"unknown role", uniqueEmail(t), "$2a$04$notarealhashbutlongenough", "Test Player", Role("REFEREE")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := repo.CreateUser(ctx, tc.email, tc.hash, tc.fullName, tc.role); err == nil {
				t.Error("CreateUser() returned nil error, want a constraint violation")
			}
		})
	}
}

func TestRepositoryUserLookups(t *testing.T) {
	repo := newTestRepository(t)
	created := createTestUser(t, repo, RoleOwner)
	ctx := context.Background()

	byEmail, err := repo.UserByEmail(ctx, created.Email)
	if err != nil {
		t.Fatalf("UserByEmail() returned error: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("UserByEmail().ID = %q, want %q", byEmail.ID, created.ID)
	}

	byID, err := repo.UserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("UserByID() returned error: %v", err)
	}
	if byID.Email != created.Email {
		t.Errorf("UserByID().Email = %q, want %q", byID.Email, created.Email)
	}
}

func TestRepositoryUserLookupMisses(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if _, err := repo.UserByEmail(ctx, "nobody@playhub.test"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UserByEmail() error = %v, want ErrUserNotFound", err)
	}
	if _, err := repo.UserByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UserByID() error = %v, want ErrUserNotFound", err)
	}
	// A subject that is not a UUID can only come from a forged token. It is a
	// missing user, not a server fault.
	if _, err := repo.UserByID(ctx, "not-a-uuid"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UserByID(non-uuid) error = %v, want ErrUserNotFound", err)
	}
}

func TestRepositoryRefreshTokenLifecycle(t *testing.T) {
	repo := newTestRepository(t)
	user := createTestUser(t, repo, RolePlayer)
	ctx := context.Background()

	expiresAt := time.Now().Add(24 * time.Hour).UTC()
	id, err := repo.CreateRefreshToken(ctx, user.ID, expiresAt)
	if err != nil {
		t.Fatalf("CreateRefreshToken() returned error: %v", err)
	}

	record, err := repo.RefreshTokenByID(ctx, id)
	if err != nil {
		t.Fatalf("RefreshTokenByID() returned error: %v", err)
	}
	if record.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", record.UserID, user.ID)
	}
	if !record.Usable(time.Now()) {
		t.Error("Usable() = false, want a fresh token to be usable")
	}

	if err := repo.RevokeRefreshToken(ctx, id); err != nil {
		t.Fatalf("RevokeRefreshToken() returned error: %v", err)
	}

	revoked, err := repo.RefreshTokenByID(ctx, id)
	if err != nil {
		t.Fatalf("RefreshTokenByID() returned error: %v", err)
	}
	if revoked.Usable(time.Now()) {
		t.Error("Usable() = true after revocation, want false")
	}
}

// Logout has to be idempotent: a client retrying it must not be told its own
// sign-out failed.
func TestRepositoryRevokeIsIdempotent(t *testing.T) {
	repo := newTestRepository(t)
	user := createTestUser(t, repo, RolePlayer)
	ctx := context.Background()

	id, err := repo.CreateRefreshToken(ctx, user.ID, time.Now().Add(time.Hour).UTC())
	if err != nil {
		t.Fatalf("CreateRefreshToken() returned error: %v", err)
	}

	for _, target := range []string{id, id, "00000000-0000-0000-0000-000000000000", "not-a-uuid"} {
		if err := repo.RevokeRefreshToken(ctx, target); err != nil {
			t.Errorf("RevokeRefreshToken(%q) returned error: %v", target, err)
		}
	}
}

func TestRepositoryRefreshTokenMisses(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if _, err := repo.RefreshTokenByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("RefreshTokenByID() error = %v, want ErrInvalidToken", err)
	}
	if _, err := repo.RefreshTokenByID(ctx, "not-a-uuid"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("RefreshTokenByID(non-uuid) error = %v, want ErrInvalidToken", err)
	}
}

// Deleting an account must take its sessions with it, or a revoked user keeps
// a working refresh token.
func TestRepositoryDeletingUserCascadesToTokens(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, uniqueEmail(t), "$2a$04$notarealhashbutlongenough", "Test Player", RolePlayer)
	if err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}

	id, err := repo.CreateRefreshToken(ctx, user.ID, time.Now().Add(time.Hour).UTC())
	if err != nil {
		t.Fatalf("CreateRefreshToken() returned error: %v", err)
	}

	if _, err := testPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, user.ID); err != nil {
		t.Fatalf("deleting the user failed: %v", err)
	}

	if _, err := repo.RefreshTokenByID(ctx, id); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("RefreshTokenByID() error = %v, want the row to have been cascaded away", err)
	}
}

// updated_at is maintained by a trigger, so a writer that forgets it still
// leaves an honest row.
func TestRepositoryUpdatedAtTrigger(t *testing.T) {
	repo := newTestRepository(t)
	user := createTestUser(t, repo, RolePlayer)
	ctx := context.Background()

	if _, err := testPool.Exec(ctx, `UPDATE users SET full_name = 'Renamed Player' WHERE id = $1`, user.ID); err != nil {
		t.Fatalf("updating the user failed: %v", err)
	}

	updated, err := repo.UserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("UserByID() returned error: %v", err)
	}
	if !updated.UpdatedAt.After(user.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want it to be later than %v", updated.UpdatedAt, user.UpdatedAt)
	}
}
