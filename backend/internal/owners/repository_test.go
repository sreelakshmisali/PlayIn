package owners

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
// the schema: unique indexes, CHECK constraints, foreign keys and cascades. A
// fake would only re-test the fake.
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

func uniqueEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%s-%d@playhub.test", sanitise(t.Name()), time.Now().UnixNano())
}

// createTestOwner inserts an OWNER account and removes it when the test ends.
// Deleting it cascades through owner_profiles and everything beneath it.
func createTestOwner(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	var userID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, '$2a$04$notarealhashbutlongenough', 'Test Owner', 'OWNER')
		RETURNING id::text`, uniqueEmail(t)).Scan(&userID)
	if err != nil {
		t.Fatalf("creating the test owner failed: %v", err)
	}

	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("cleaning up user %s failed: %v", userID, err)
		}
	})
	return userID
}

// createTestOwnerProfile creates an owner account with a profile and returns
// the profile id, the anchor turfs are created against.
func createTestOwnerProfile(t *testing.T, repo *Repository) string {
	t.Helper()

	userID := createTestOwner(t)
	profile, _, err := repo.SaveProfile(context.Background(), userID, profileFields{DisplayName: "Test Arena"})
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	// SaveProfile returns the projection, not the surrogate id; resolve it the
	// same way the service does.
	profileID, err := repo.ProfileIDForUser(context.Background(), profile.UserID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}
	return profileID
}

func sportByName(t *testing.T, name string) string {
	t.Helper()

	var id string
	err := testPool.QueryRow(context.Background(), `SELECT id::text FROM sports WHERE name = $1`, name).Scan(&id)
	if err != nil {
		t.Fatalf("looking up sport %q failed: %v", name, err)
	}
	return id
}

func amenityByName(t *testing.T, name string) string {
	t.Helper()

	var id string
	err := testPool.QueryRow(context.Background(), `SELECT id::text FROM amenities WHERE name = $1`, name).Scan(&id)
	if err != nil {
		t.Fatalf("looking up amenity %q failed: %v", name, err)
	}
	return id
}

func testTurfFields() turfFields {
	lat, lng := 9.9312, 76.2673
	capacity := int32(22)
	return turfFields{
		Name: "Riverside Turf", Description: "Nice turf.",
		Address: "123 River Road", City: "Kochi",
		Latitude: &lat, Longitude: &lng, Capacity: &capacity,
		OpeningTime: "06:00", ClosingTime: "22:00",
	}
}

// --- owner profile -------------------------------------------------------------

func TestRepositoryOwnerProfileUpsert(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestOwner(t)
	ctx := context.Background()

	created, inserted, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Test Arena", Phone: "+91 98765 43210"})
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if !inserted {
		t.Error("inserted = false on the first save, want true")
	}
	if created.UserID != userID {
		t.Errorf("UserID = %q, want %q", created.UserID, userID)
	}

	_, inserted, err = repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Renamed Arena"})
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if inserted {
		t.Error("inserted = true on the second save, want false")
	}
}

func TestRepositoryProfileByUserID(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestOwner(t)
	ctx := context.Background()

	if _, err := repo.ProfileByUserID(ctx, userID); !errors.Is(err, ErrOwnerProfileNotFound) {
		t.Errorf("ProfileByUserID() before a save error = %v, want ErrOwnerProfileNotFound", err)
	}

	if _, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Test Arena", Phone: "+91 98765 43210"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}

	got, err := repo.ProfileByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ProfileByUserID() returned error: %v", err)
	}
	if got.DisplayName != "Test Arena" || got.Phone != "+91 98765 43210" {
		t.Errorf("profile = %+v, want the saved values", got)
	}

	if _, err := repo.ProfileByUserID(ctx, "not-a-uuid"); !errors.Is(err, ErrOwnerProfileNotFound) {
		t.Errorf("ProfileByUserID(non-uuid) error = %v, want ErrOwnerProfileNotFound", err)
	}
}

func TestRepositoryOwnerProfileRejectsBadInput(t *testing.T) {
	repo := newTestRepository(t)
	userID := createTestOwner(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		fields profileFields
	}{
		{"blank display name", profileFields{DisplayName: " "}},
		{"overlong display name", profileFields{DisplayName: repeatStr("a", 121)}},
		{"letters in phone", profileFields{DisplayName: "Arena", Phone: "call-me"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := repo.SaveProfile(ctx, userID, tc.fields); err == nil {
				t.Error("SaveProfile() returned nil error, want a constraint violation")
			}
		})
	}
}

func repeatStr(s string, n int) string {
	out := make([]byte, 0, n)
	for range n {
		out = append(out, s...)
	}
	return string(out)
}

// --- turf CRUD -----------------------------------------------------------------

func TestRepositoryCreateTurfStartsAsDraft(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)

	turf, err := repo.CreateTurf(context.Background(), ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	if turf.Status != StatusDraft {
		t.Errorf("Status = %q, want DRAFT", turf.Status)
	}
	if turf.OwnerDisplayName != "Test Arena" {
		t.Errorf("OwnerDisplayName = %q, want Test Arena", turf.OwnerDisplayName)
	}
}

func TestRepositoryCreateTurfRejectsDuplicateNamePerOwner(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	if _, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields()); err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	if _, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields()); !errors.Is(err, ErrTurfNameTaken) {
		t.Errorf("CreateTurf() error = %v, want ErrTurfNameTaken", err)
	}
}

// The unique index is scoped per owner: it is (owner_id, lower(name)), not a
// bare unique on name, so two different owners can use the same turf name.
func TestRepositoryTurfNameUniqueIsPerOwnerNotGlobal(t *testing.T) {
	repo := newTestRepository(t)
	first := createTestOwnerProfile(t, repo)
	second := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	if _, err := repo.CreateTurf(ctx, first, testTurfFields()); err != nil {
		t.Fatalf("CreateTurf() for the first owner returned error: %v", err)
	}
	if _, err := repo.CreateTurf(ctx, second, testTurfFields()); err != nil {
		t.Errorf("CreateTurf() for a different owner with the same name returned error: %v", err)
	}
}

func TestRepositoryCreateTurfRejectsBadInput(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	tests := []struct {
		name   string
		mutate func(*turfFields)
	}{
		{"blank name", func(f *turfFields) { f.Name = " " }},
		{"short address", func(f *turfFields) { f.Address = "123" }},
		{"bad opening time", func(f *turfFields) { f.OpeningTime = "6am" }},
		{"latitude without longitude", func(f *turfFields) { f.Longitude = nil }},
		{"latitude out of range", func(f *turfFields) { v := 200.0; f.Latitude = &v }},
		{"negative capacity", func(f *turfFields) { v := int32(-1); f.Capacity = &v }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := testTurfFields()
			tc.mutate(&f)
			if _, err := repo.CreateTurf(ctx, ownerProfileID, f); err == nil {
				t.Error("CreateTurf() returned nil error, want a constraint violation")
			}
		})
	}
}

// The status CHECK constraint, exercised directly since the repository never
// writes an arbitrary status itself; every status the code sets is one of the
// five literals baked into its own queries.
func TestRepositoryStatusCheckConstraint(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if _, err := testPool.Exec(ctx, `UPDATE turfs SET status = 'LIVE' WHERE id = $1`, turf.ID); err == nil {
		t.Error("setting an unknown status succeeded, want the CHECK constraint to refuse it")
	}
}

func TestRepositoryTurfsByOwner(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	other := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	if _, err := repo.CreateTurf(ctx, mine, testTurfFields()); err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	second := testTurfFields()
	second.Name = "Second Turf"
	if _, err := repo.CreateTurf(ctx, mine, second); err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	if _, err := repo.CreateTurf(ctx, other, testTurfFields()); err != nil {
		t.Fatalf("CreateTurf() for the other owner returned error: %v", err)
	}

	mineList, err := repo.TurfsByOwner(ctx, mine)
	if err != nil {
		t.Fatalf("TurfsByOwner() returned error: %v", err)
	}
	if len(mineList) != 2 {
		t.Fatalf("TurfsByOwner() = %d turfs, want 2, not the other owner's", len(mineList))
	}
	for _, turf := range mineList {
		if turf.OwnerDisplayName != "Test Arena" {
			t.Errorf("OwnerDisplayName = %q, want Test Arena", turf.OwnerDisplayName)
		}
	}

	empty, err := repo.TurfsByOwner(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("TurfsByOwner() for an unknown owner returned error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("TurfsByOwner() for an unknown owner = %d turfs, want 0", len(empty))
	}
}

func TestRepositoryTurfByOwnerAndIDNotFoundCoversWrongOwner(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	theirs := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, theirs, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if _, err := repo.TurfByOwnerAndID(ctx, mine, turf.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("TurfByOwnerAndID() across owners error = %v, want ErrTurfNotFound", err)
	}
	if _, err := repo.TurfByOwnerAndID(ctx, mine, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("TurfByOwnerAndID() unknown id error = %v, want ErrTurfNotFound", err)
	}
	if _, err := repo.TurfByOwnerAndID(ctx, mine, "not-a-uuid"); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("TurfByOwnerAndID() non-uuid error = %v, want ErrTurfNotFound", err)
	}
}

func TestRepositoryUpdateTurfRevertsApprovedToPending(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	if _, err := testPool.Exec(ctx, `UPDATE turfs SET status = 'APPROVED' WHERE id = $1`, turf.ID); err != nil {
		t.Fatalf("forcing APPROVED failed: %v", err)
	}

	f := testTurfFields()
	f.Name = "Renamed"
	updated, err := repo.UpdateTurf(ctx, ownerProfileID, turf.ID, f)
	if err != nil {
		t.Fatalf("UpdateTurf() returned error: %v", err)
	}
	if updated.Status != StatusPendingApproval {
		t.Errorf("Status = %q, want PENDING_APPROVAL", updated.Status)
	}
}

func TestRepositoryUpdateTurfCannotReachAnotherOwnersTurf(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	theirs := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, theirs, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	f := testTurfFields()
	f.Name = "Hijacked"
	if _, err := repo.UpdateTurf(ctx, mine, turf.ID, f); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("UpdateTurf() across owners error = %v, want ErrTurfNotFound", err)
	}

	unchanged, err := repo.TurfByOwnerAndID(ctx, theirs, turf.ID)
	if err != nil {
		t.Fatalf("TurfByOwnerAndID() returned error: %v", err)
	}
	if unchanged.Name != "Riverside Turf" {
		t.Errorf("Name = %q, want it unchanged", unchanged.Name)
	}
}

func TestRepositoryDeleteTurfCannotReachAnotherOwnersTurf(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	theirs := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, theirs, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if err := repo.DeleteTurf(ctx, mine, turf.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("DeleteTurf() across owners error = %v, want ErrTurfNotFound", err)
	}
	if _, err := repo.TurfByOwnerAndID(ctx, theirs, turf.ID); err != nil {
		t.Errorf("the turf was deleted by a non-owner: %v", err)
	}
}

func TestRepositoryDeletingOwnerCascadesToTurfs(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	userID := createTestOwner(t)
	profile, _, err := repo.SaveProfile(ctx, userID, profileFields{DisplayName: "Cascade Arena"})
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	ownerProfileID, err := repo.ProfileIDForUser(ctx, profile.UserID)
	if err != nil {
		t.Fatalf("ProfileIDForUser() returned error: %v", err)
	}

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	football := sportByName(t, "Football")
	if err := repo.SetTurfSport(ctx, ownerProfileID, turf.ID, football); err != nil {
		t.Fatalf("SetTurfSport() returned error: %v", err)
	}

	if _, err := testPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Fatalf("deleting the owner failed: %v", err)
	}

	var turfCount, sportCount int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM turfs WHERE id = $1`, turf.ID).Scan(&turfCount); err != nil {
		t.Fatalf("counting turfs failed: %v", err)
	}
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM turf_sports WHERE turf_id = $1`, turf.ID).Scan(&sportCount); err != nil {
		t.Fatalf("counting turf_sports failed: %v", err)
	}
	if turfCount != 0 || sportCount != 0 {
		t.Errorf("turf count = %d, turf_sports count = %d, want both cascaded to 0", turfCount, sportCount)
	}
}

// --- submit ----------------------------------------------------------------------

func TestRepositorySubmitTurf(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	submitted, err := repo.SubmitTurf(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}
	if submitted.Status != StatusPendingApproval {
		t.Errorf("Status = %q, want PENDING_APPROVAL", submitted.Status)
	}
}

func TestRepositorySubmitTurfRejectsWrongStatus(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	if _, err := testPool.Exec(ctx, `UPDATE turfs SET status = 'APPROVED' WHERE id = $1`, turf.ID); err != nil {
		t.Fatalf("forcing APPROVED failed: %v", err)
	}

	if _, err := repo.SubmitTurf(ctx, ownerProfileID, turf.ID); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("SubmitTurf() error = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestRepositorySubmitTurfCannotReachAnotherOwnersTurf(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	theirs := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, theirs, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if _, err := repo.SubmitTurf(ctx, mine, turf.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("SubmitTurf() across owners error = %v, want ErrTurfNotFound", err)
	}
}

// --- turf sports and amenities: the guarded insert enforces membership at the
// database, not just in Go ---------------------------------------------------

func TestRepositorySetTurfSportEnforcesOwnershipAndValidityAtTheDatabase(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	theirs := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	myTurf, err := repo.CreateTurf(ctx, mine, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	theirTurf, err := repo.CreateTurf(ctx, theirs, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	football := sportByName(t, "Football")

	if err := repo.SetTurfSport(ctx, mine, myTurf.ID, football); err != nil {
		t.Errorf("SetTurfSport() on my own turf returned error: %v", err)
	}
	if err := repo.SetTurfSport(ctx, mine, theirTurf.ID, football); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("SetTurfSport() on another owner's turf error = %v, want ErrTurfNotFound", err)
	}
	if err := repo.SetTurfSport(ctx, mine, myTurf.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrSportNotFound) {
		t.Errorf("SetTurfSport() unknown sport error = %v, want ErrSportNotFound", err)
	}

	// The other owner's turf must not have gained the sport from the rejected
	// attempt.
	untouched, err := repo.TurfByOwnerAndID(ctx, theirs, theirTurf.ID)
	if err != nil {
		t.Fatalf("TurfByOwnerAndID() returned error: %v", err)
	}
	if len(untouched.Sports) != 0 {
		t.Errorf("Sports = %+v, want none, the attach was supposed to be refused", untouched.Sports)
	}
}

func TestRepositorySetTurfSportIsIdempotent(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	football := sportByName(t, "Football")

	for i := 0; i < 2; i++ {
		if err := repo.SetTurfSport(ctx, ownerProfileID, turf.ID, football); err != nil {
			t.Fatalf("SetTurfSport() call %d returned error: %v", i, err)
		}
	}

	got, err := repo.TurfByOwnerAndID(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("TurfByOwnerAndID() returned error: %v", err)
	}
	if len(got.Sports) != 1 {
		t.Errorf("Sports = %+v, want exactly 1 after two identical calls", got.Sports)
	}
}

func TestRepositoryRemoveTurfSportCannotReachAnotherOwnersTurf(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	theirs := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	theirTurf, err := repo.CreateTurf(ctx, theirs, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	football := sportByName(t, "Football")
	if err := repo.SetTurfSport(ctx, theirs, theirTurf.ID, football); err != nil {
		t.Fatalf("SetTurfSport() returned error: %v", err)
	}

	if err := repo.RemoveTurfSport(ctx, mine, theirTurf.ID, football); !errors.Is(err, ErrTurfSportNotFound) {
		t.Errorf("RemoveTurfSport() across owners error = %v, want ErrTurfSportNotFound", err)
	}

	still, err := repo.TurfByOwnerAndID(ctx, theirs, theirTurf.ID)
	if err != nil {
		t.Fatalf("TurfByOwnerAndID() returned error: %v", err)
	}
	if len(still.Sports) != 1 {
		t.Error("the sport was removed by a non-owner")
	}
}

// A sport a turf has chosen cannot be deleted out from under it.
func TestRepositorySportDeleteIsRestrictedByTurfUsage(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	tennisID := sportByName(t, "Tennis")
	if err := repo.SetTurfSport(ctx, ownerProfileID, turf.ID, tennisID); err != nil {
		t.Fatalf("SetTurfSport() returned error: %v", err)
	}

	if _, err := testPool.Exec(ctx, `DELETE FROM sports WHERE id = $1`, tennisID); err == nil {
		t.Error("deleting a chosen sport succeeded, want it restricted by the foreign key")
	}
}

func TestRepositoryTurfAmenityLifecycle(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	parking := amenityByName(t, "Parking")

	if err := repo.SetTurfAmenity(ctx, ownerProfileID, turf.ID, parking); err != nil {
		t.Fatalf("SetTurfAmenity() returned error: %v", err)
	}
	got, err := repo.TurfByOwnerAndID(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("TurfByOwnerAndID() returned error: %v", err)
	}
	if len(got.Amenities) != 1 || got.Amenities[0].Slug != "parking" {
		t.Fatalf("Amenities = %+v, want one parking entry", got.Amenities)
	}

	if err := repo.RemoveTurfAmenity(ctx, ownerProfileID, turf.ID, parking); err != nil {
		t.Fatalf("RemoveTurfAmenity() returned error: %v", err)
	}
	got, err = repo.TurfByOwnerAndID(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("TurfByOwnerAndID() returned error: %v", err)
	}
	if len(got.Amenities) != 0 {
		t.Error("the amenity is still attached after removal")
	}
}

// --- turf images -----------------------------------------------------------------

func TestRepositoryTurfImageLifecycle(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	img, err := repo.AddTurfImage(ctx, ownerProfileID, turf.ID, "https://cdn.playhub.test/1.jpg")
	if err != nil {
		t.Fatalf("AddTurfImage() returned error: %v", err)
	}

	if err := repo.RemoveTurfImage(ctx, ownerProfileID, turf.ID, img.ID); err != nil {
		t.Fatalf("RemoveTurfImage() returned error: %v", err)
	}

	got, err := repo.TurfByOwnerAndID(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("TurfByOwnerAndID() returned error: %v", err)
	}
	if len(got.Images) != 0 {
		t.Error("the image is still attached after removal")
	}
}

func TestRepositoryAddTurfImageRejectsScriptURL(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if _, err := repo.AddTurfImage(ctx, ownerProfileID, turf.ID, "javascript:alert(1)"); err == nil {
		t.Error("AddTurfImage() with a javascript: URL returned nil error, want the CHECK constraint to refuse it")
	}
}

func TestRepositoryAddTurfImageCannotReachAnotherOwnersTurf(t *testing.T) {
	repo := newTestRepository(t)
	mine := createTestOwnerProfile(t, repo)
	theirs := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	theirTurf, err := repo.CreateTurf(ctx, theirs, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if _, err := repo.AddTurfImage(ctx, mine, theirTurf.ID, "https://cdn.playhub.test/x.jpg"); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("AddTurfImage() across owners error = %v, want ErrTurfNotFound", err)
	}
}

func TestRepositoryAddTurfImageEnforcesCap(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	for i := 0; i < maxTurfImages; i++ {
		url := fmt.Sprintf("https://cdn.playhub.test/%d.jpg", i)
		if _, err := repo.AddTurfImage(ctx, ownerProfileID, turf.ID, url); err != nil {
			t.Fatalf("AddTurfImage() #%d returned error: %v", i, err)
		}
	}

	if _, err := repo.AddTurfImage(ctx, ownerProfileID, turf.ID, "https://cdn.playhub.test/over.jpg"); !errors.Is(err, ErrTooManyImages) {
		t.Errorf("AddTurfImage() over the cap error = %v, want ErrTooManyImages", err)
	}
}

// --- amenities catalogue -----------------------------------------------------

// The six amenities are seeded by migration 000004, not by the application,
// so this is the test that the migration's INSERT actually ran.
func TestRepositorySeededAmenities(t *testing.T) {
	repo := newTestRepository(t)

	amenities, err := repo.Amenities(context.Background())
	if err != nil {
		t.Fatalf("Amenities() returned error: %v", err)
	}
	if len(amenities) != 6 {
		t.Fatalf("Amenities() returned %d amenities, want the 6 seeded ones", len(amenities))
	}

	want := []string{"Cafeteria", "Changing Room", "Drinking Water", "Floodlights", "Parking", "Washroom"}
	for i, name := range want {
		if amenities[i].Name != name {
			t.Errorf("amenities[%d] = %q, want %q", i, amenities[i].Name, name)
		}
	}
}

// --- public visibility -----------------------------------------------------------

func TestRepositoryPublicTurfsOnlyReturnsApproved(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	for _, status := range []Status{"DRAFT", "PENDING_APPROVAL", "REJECTED", "SUSPENDED"} {
		if _, err := testPool.Exec(ctx, `UPDATE turfs SET status = $2 WHERE id = $1`, turf.ID, string(status)); err != nil {
			t.Fatalf("setting status %s failed: %v", status, err)
		}

		if _, err := repo.PublicTurfByID(ctx, turf.ID); !errors.Is(err, ErrTurfNotFound) {
			t.Errorf("PublicTurfByID() with status %s error = %v, want ErrTurfNotFound", status, err)
		}
		list, err := repo.PublicTurfs(ctx)
		if err != nil {
			t.Fatalf("PublicTurfs() returned error: %v", err)
		}
		for _, listed := range list {
			if listed.ID == turf.ID {
				t.Errorf("PublicTurfs() with status %s includes the turf, want it excluded", status)
			}
		}
	}

	if _, err := testPool.Exec(ctx, `UPDATE turfs SET status = 'APPROVED' WHERE id = $1`, turf.ID); err != nil {
		t.Fatalf("approving the turf failed: %v", err)
	}

	got, err := repo.PublicTurfByID(ctx, turf.ID)
	if err != nil {
		t.Fatalf("PublicTurfByID() once approved returned error: %v", err)
	}
	if got.ID != turf.ID {
		t.Errorf("ID = %q, want %q", got.ID, turf.ID)
	}
}
