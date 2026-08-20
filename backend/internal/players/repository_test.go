package players

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
// the schema: the seeded catalogue, the unique index that keeps one profile per
// account, the CHECK constraints, and the guarded insert that enforces position
// membership. A fake would only re-test the fake.
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

// createTestPlayer inserts an account to hang a profile off, and removes it when
// the test ends. The cascade takes the profile and its sports with it.
func createTestPlayer(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	email := fmt.Sprintf("player-%d-%s@playhub.test", time.Now().UnixNano(), sanitise(t.Name()))

	var userID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, '$2a$04$notarealhashbutlongenough', 'Test Player', 'PLAYER')
		RETURNING id::text`, email).Scan(&userID)
	if err != nil {
		t.Fatalf("creating the test user failed: %v", err)
	}

	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("cleaning up user %s failed: %v", userID, err)
		}
	})

	return userID
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

// sportByName finds a seeded sport. It also proves the seed landed.
func sportByName(t *testing.T, repo *Repository, name string) Sport {
	t.Helper()

	sports, err := repo.Sports(context.Background())
	if err != nil {
		t.Fatalf("Sports() returned error: %v", err)
	}
	for _, s := range sports {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("sport %q is not in the catalogue", name)
	return Sport{}
}

// The six sports are seeded by migration 000003, not by the application, so
// this is the test that the migration actually ran its INSERT.
func TestRepositorySeededCatalogue(t *testing.T) {
	repo := newTestRepository(t)

	sports, err := repo.Sports(context.Background())
	if err != nil {
		t.Fatalf("Sports() returned error: %v", err)
	}
	if len(sports) != 6 {
		t.Fatalf("Sports() returned %d sports, want the 6 seeded ones", len(sports))
	}

	// Ordered by name, so the client does not have to sort.
	want := []string{"Badminton", "Basketball", "Cricket", "Football", "Tennis", "Volleyball"}
	for i, name := range want {
		if sports[i].Name != name {
			t.Errorf("sports[%d] = %q, want %q", i, sports[i].Name, name)
		}
	}
}

// Positions are what make "where applicable" a property of the data.
func TestRepositorySeededPositions(t *testing.T) {
	repo := newTestRepository(t)

	byName := map[string]Sport{}
	sports, err := repo.Sports(context.Background())
	if err != nil {
		t.Fatalf("Sports() returned error: %v", err)
	}
	for _, s := range sports {
		byName[s.Name] = s
	}

	tests := []struct {
		sport string
		count int
	}{
		{"Football", 4},
		{"Cricket", 4},
		{"Basketball", 5},
		{"Volleyball", 5},
		{"Badminton", 0},
		{"Tennis", 0},
	}

	for _, tc := range tests {
		if got := len(byName[tc.sport].Positions); got != tc.count {
			t.Errorf("%s has %d positions, want %d", tc.sport, got, tc.count)
		}
	}
	if byName["Badminton"].Positions == nil {
		t.Error("Badminton positions = nil, want an empty array")
	}
}

func TestRepositorySportByID(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	football := sportByName(t, repo, "Football")

	got, err := repo.SportByID(ctx, football.ID)
	if err != nil {
		t.Fatalf("SportByID() returned error: %v", err)
	}
	if got.Slug != "football" {
		t.Errorf("Slug = %q, want football", got.Slug)
	}

	if _, err := repo.SportByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrSportNotFound) {
		t.Errorf("SportByID(unknown) error = %v, want ErrSportNotFound", err)
	}
	// A non-UUID can only come from a bad request, not a server fault.
	if _, err := repo.SportByID(ctx, "not-a-uuid"); !errors.Is(err, ErrSportNotFound) {
		t.Errorf("SportByID(non-uuid) error = %v, want ErrSportNotFound", err)
	}
}

func TestRepositorySaveProfile(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	fields := profileFields{
		DisplayName: "Priya Raman",
		ImageURL:    "https://cdn.playhub.test/p.jpg",
		Bio:         "Weekend midfielder.",
		Location:    "Kochi",
	}

	created, inserted, err := repo.SaveProfile(ctx, userID, fields)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if !inserted {
		t.Error("inserted = false on the first save, want true")
	}
	if created.UserID != userID {
		t.Errorf("UserID = %q, want %q", created.UserID, userID)
	}
	if created.Sports == nil {
		t.Error("Sports is nil, want an empty slice")
	}

	// The second save updates rather than failing on the unique index, which is
	// what keeps one profile per account safe under a concurrent first write.
	_, inserted, err = repo.SaveProfile(ctx, userID, fields)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if inserted {
		t.Error("inserted = true on the second save, want false")
	}
}

// Empty strings become NULL, and come back as empty strings. Storing "" and
// NULL as different states would give two ways to say "not set".
func TestRepositorySaveProfileNullsEmptyOptionals(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	saved, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"})
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if saved.Bio != "" || saved.Location != "" || saved.ImageURL != "" {
		t.Errorf("profile = %+v, want the optionals empty", saved)
	}

	var nulls int
	err = testPool.QueryRow(ctx, `
		SELECT count(*) FROM player_profiles
		WHERE user_id = $1 AND bio IS NULL AND location IS NULL AND image_url IS NULL`, userID).Scan(&nulls)
	if err != nil {
		t.Fatalf("counting nulls failed: %v", err)
	}
	if nulls != 1 {
		t.Error("the optional columns were stored as empty strings, want NULL")
	}
}

func TestRepositorySaveProfileRejectsBadInput(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		fields profileFields
	}{
		{"blank display name", profileFields{DisplayName: " "}},
		{"one character display name", profileFields{DisplayName: "P"}},
		{"overlong display name", profileFields{DisplayName: repeat("a", 81)}},
		{"overlong bio", profileFields{DisplayName: "Priya", Bio: repeat("a", 501)}},
		{"one character location", profileFields{DisplayName: "Priya", Location: "K"}},
		// The database refuses a script URL even if a caller reaches it around
		// the validator.
		{"script image url", profileFields{DisplayName: "Priya", ImageURL: "javascript:alert(1)"}},
		{"image url with a space", profileFields{DisplayName: "Priya", ImageURL: "https://cdn.test/a b"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := repo.SaveProfile(ctx, userID, tc.fields); err == nil {
				t.Error("SaveProfile() returned nil error, want a constraint violation")
			}
		})
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, n)
	for range n {
		out = append(out, s...)
	}
	return string(out)
}

func TestRepositoryProfileNotFound(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if _, err := repo.ProfileByUserID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("ProfileByUserID() error = %v, want ErrProfileNotFound", err)
	}
	if _, err := repo.ProfileByUserID(ctx, "not-a-uuid"); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("ProfileByUserID(non-uuid) error = %v, want ErrProfileNotFound", err)
	}
}

func TestRepositorySetSport(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	profileID, err := repo.ProfileIDForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}

	football := sportByName(t, repo, "Football")
	if err := repo.SetSport(ctx, profileID, football.ID, "Midfielder"); err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}

	profile, err := repo.ProfileByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileByUserID() returned error: %v", err)
	}
	if len(profile.Sports) != 1 {
		t.Fatalf("Sports = %v, want 1 entry", profile.Sports)
	}
	if profile.Sports[0].Position != "Midfielder" || profile.Sports[0].Sport.Name != "Football" {
		t.Errorf("Sports[0] = %+v, want Football/Midfielder", profile.Sports[0])
	}
}

// The insert selects from sports, so membership is checked against the live row
// inside the statement. This is the test that the guard is real and not just a
// Go-side check that could be bypassed.
func TestRepositorySetSportEnforcesPositionAtTheDatabase(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	profileID, err := repo.ProfileIDForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}

	football := sportByName(t, repo, "Football")
	cricket := sportByName(t, repo, "Cricket")
	badminton := sportByName(t, repo, "Badminton")

	tests := []struct {
		name     string
		sportID  string
		position string
	}{
		{"invented position", football.ID, "Striker"},
		{"position from another sport", football.ID, "Wicketkeeper"},
		{"position on a sport with none", badminton.ID, "Smasher"},
		{"unknown sport", "00000000-0000-0000-0000-000000000000", ""},
		{"non-uuid sport", "not-a-uuid", ""},
		{"cricket position on cricket is fine, this one is not", cricket.ID, "Goalkeeper"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := repo.SetSport(ctx, profileID, tc.sportID, tc.position); !errors.Is(err, ErrSportNotFound) {
				t.Errorf("SetSport() error = %v, want ErrSportNotFound", err)
			}
		})
	}

	// Nothing was written by any of the refusals.
	profile, err := repo.ProfileByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileByUserID() returned error: %v", err)
	}
	if len(profile.Sports) != 0 {
		t.Errorf("Sports = %v, want none written by the refused calls", profile.Sports)
	}
}

// Sports with no positions can still be chosen, just without one.
func TestRepositorySetSportWithoutPosition(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	profileID, err := repo.ProfileIDForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}

	badminton := sportByName(t, repo, "Badminton")
	if err := repo.SetSport(ctx, profileID, badminton.ID, ""); err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}

	profile, err := repo.ProfileByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileByUserID() returned error: %v", err)
	}
	if profile.Sports[0].Position != "" {
		t.Errorf("Position = %q, want empty", profile.Sports[0].Position)
	}

	// Stored as NULL, not as an empty string.
	var isNull bool
	err = testPool.QueryRow(ctx, `
		SELECT position IS NULL FROM player_sports WHERE profile_id = $1`, profileID).Scan(&isNull)
	if err != nil {
		t.Fatalf("reading the stored position failed: %v", err)
	}
	if !isNull {
		t.Error("the absent position was stored as an empty string, want NULL")
	}
}

// The unique index on (profile_id, sport_id) is the conflict target, so a
// repeat changes the position instead of adding a duplicate row.
func TestRepositorySetSportUpserts(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	profileID, err := repo.ProfileIDForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}

	football := sportByName(t, repo, "Football")
	for _, position := range []string{"Defender", "Forward"} {
		if err := repo.SetSport(ctx, profileID, football.ID, position); err != nil {
			t.Fatalf("SetSport(%q) returned error: %v", position, err)
		}
	}

	profile, err := repo.ProfileByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileByUserID() returned error: %v", err)
	}
	if len(profile.Sports) != 1 {
		t.Fatalf("Sports = %v, want 1 row after a repeat", profile.Sports)
	}
	if profile.Sports[0].Position != "Forward" {
		t.Errorf("Position = %q, want Forward", profile.Sports[0].Position)
	}
}

func TestRepositoryRemoveSport(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	profileID, err := repo.ProfileIDForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}

	football := sportByName(t, repo, "Football")
	cricket := sportByName(t, repo, "Cricket")

	if err := repo.SetSport(ctx, profileID, football.ID, "Defender"); err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}
	if err := repo.RemoveSport(ctx, profileID, football.ID); err != nil {
		t.Fatalf("RemoveSport() returned error: %v", err)
	}

	// Removing one the player does not have is reported, not swallowed.
	if err := repo.RemoveSport(ctx, profileID, cricket.ID); !errors.Is(err, ErrSportNotPreferred) {
		t.Errorf("RemoveSport(not chosen) error = %v, want ErrSportNotPreferred", err)
	}
	if err := repo.RemoveSport(ctx, profileID, "not-a-uuid"); !errors.Is(err, ErrSportNotPreferred) {
		t.Errorf("RemoveSport(non-uuid) error = %v, want ErrSportNotPreferred", err)
	}
}

// Deleting an account must take its profile and preferred sports with it.
func TestRepositoryDeletingUserCascades(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	var userID string
	email := fmt.Sprintf("cascade-%d@playhub.test", time.Now().UnixNano())
	err := testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, '$2a$04$notarealhashbutlongenough', 'Test Player', 'PLAYER')
		RETURNING id::text`, email).Scan(&userID)
	if err != nil {
		t.Fatalf("creating the test user failed: %v", err)
	}

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	profileID, err := repo.ProfileIDForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}
	if err := repo.SetSport(ctx, profileID, sportByName(t, repo, "Tennis").ID, ""); err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}

	if _, err := testPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("deleting the user failed: %v", err)
	}

	if _, err := repo.ProfileByUserID(ctx, userID); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("ProfileByUserID() error = %v, want the profile cascaded away", err)
	}

	var orphans int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM player_sports WHERE profile_id = $1`, profileID).Scan(&orphans); err != nil {
		t.Fatalf("counting orphaned sports failed: %v", err)
	}
	if orphans != 0 {
		t.Errorf("player_sports rows = %d, want them cascaded away", orphans)
	}
}

// A sport players have chosen cannot be deleted out from under them. Retiring
// it with is_active is the supported move.
func TestRepositorySportDeleteIsRestricted(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	profileID, err := repo.ProfileIDForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}

	tennis := sportByName(t, repo, "Tennis")
	if err := repo.SetSport(ctx, profileID, tennis.ID, ""); err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}

	if _, err := testPool.Exec(ctx, `DELETE FROM sports WHERE id = $1`, tennis.ID); err == nil {
		t.Error("deleting a chosen sport succeeded, want it restricted by the foreign key")
	}
}

// updated_at is maintained by a trigger, so a writer that forgets it still
// leaves an honest row.
func TestRepositoryUpdatedAtTrigger(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestPlayer(t)
	ctx := context.Background()

	first, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya"})
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}

	second, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Priya R"})
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}

	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want it later than %v", second.UpdatedAt, first.UpdatedAt)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("CreatedAt changed on update: %v then %v", first.CreatedAt, second.CreatedAt)
	}
}
