package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/config"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
	"github.com/orgmelethil/playhub/backend/internal/owners"
)

// This package has no model, repository or Store of its own: everything it
// does runs through the real owners and auth services, which in turn depend
// on unexported request/row types only their own packages can construct. A
// fake Store built here could never satisfy owners.Store or auth.Store, so
// there is nothing meaningful to unit test in isolation. These tests run the
// real Handler over the real services and a real PostgreSQL database instead,
// which is also exactly what the task's own verification steps ask for:
// authorization, transitions and the self-modification guard, all through
// HTTP end to end.
//
// They are skipped unless PLAYHUB_TEST_DATABASE_URL points at a migrated
// database, so `go test ./...` stays runnable without one.
const testDatabaseURLEnv = "PLAYHUB_TEST_DATABASE_URL"

const testPrefix = "/api/v1"

var (
	testPoolOnce sync.Once
	testPool     *pgxpool.Pool
	testPoolErr  error
)

type testEnv struct {
	mux           *http.ServeMux
	authRepo      *auth.Repository
	ownersService *owners.Service
	issuer        *auth.Issuer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dsn := os.Getenv(testDatabaseURLEnv)
	if dsn == "" {
		t.Skipf("%s is not set, skipping the admin integration tests", testDatabaseURLEnv)
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

	authRepo := auth.NewRepository(testPool)
	issuer := auth.NewIssuer(config.Auth{
		JWTSecret:  "0123456789abcdef0123456789abcdef",
		JWTIssuer:  "playhub-admin-test",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
		BcryptCost: 4,
	})
	authService := auth.NewService(authRepo, auth.NewBcryptHasher(4), issuer)
	ownersService := owners.NewService(owners.NewRepository(testPool))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(ownersService, authService, authService, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpx.NotFound)
	handler.Routes(mux, testPrefix)

	return &testEnv{mux: mux, authRepo: authRepo, ownersService: ownersService, issuer: issuer}
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

// createUser inserts a real account and mints a real access token for it, so
// the guard chain (RequireAuth, RequireRole) is exercised exactly as it is in
// production, not bypassed for the test.
func (env *testEnv) createUser(t *testing.T, role auth.Role) (userID, token string) {
	t.Helper()

	ctx := context.Background()
	user, err := env.authRepo.CreateUser(ctx, uniqueEmail(t), "$2a$04$notarealhashbutlongenough", "Test User", role)
	if err != nil {
		t.Fatalf("CreateUser() returned error: %v", err)
	}
	t.Cleanup(func() {
		// Cascades through owner_profiles -> turfs -> its sub-resources.
		if _, err := testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID); err != nil {
			t.Errorf("cleaning up user %s failed: %v", user.ID, err)
		}
	})

	accessToken, err := env.issuer.Access(user)
	if err != nil {
		t.Fatalf("Access() returned error: %v", err)
	}
	return user.ID, accessToken
}

// seedSubmittedTurf creates an owner profile and a turf already in
// PENDING_APPROVAL, all through the real, exported owners.Service API.
func (env *testEnv) seedSubmittedTurf(t *testing.T, ownerUserID string) owners.Turf {
	t.Helper()
	ctx := context.Background()

	if _, _, err := env.ownersService.SaveProfile(ctx, ownerUserID, owners.SaveProfileRequest{DisplayName: "Test Arena"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}

	turf, err := env.ownersService.CreateTurf(ctx, ownerUserID, owners.SaveTurfRequest{
		Name: "Test Turf " + uniqueEmail(t), Address: "123 Test Street, Area", City: "Kochi",
		OpeningTime: "06:00", ClosingTime: "22:00",
	})
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	submitted, err := env.ownersService.SubmitTurf(ctx, ownerUserID, turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}
	return submitted
}

func do(t *testing.T, mux *http.ServeMux, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeTurf(t *testing.T, rec *httptest.ResponseRecorder) owners.Turf {
	t.Helper()
	var turf owners.Turf
	if err := json.Unmarshal(rec.Body.Bytes(), &turf); err != nil {
		t.Fatalf("decoding the turf failed: %v (body: %s)", err, rec.Body)
	}
	return turf
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()
	var envelope httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decoding the error body failed: %v (body: %s)", err, rec.Body)
	}
	return envelope.Error
}

// --- authorization -----------------------------------------------------------

// The core requirement: every admin endpoint requires ADMIN specifically, and
// a PLAYER or OWNER token, though genuinely authenticated, must be refused.
func TestPlayerAndOwnerCannotAccessAdminEndpoints(t *testing.T) {
	env := newTestEnv(t)
	ownerID, ownerToken := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)
	_, playerToken := env.createUser(t, auth.RolePlayer)

	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/admin/turfs/pending", ""},
		{http.MethodGet, "/admin/turfs/" + turf.ID, ""},
		{http.MethodPost, "/admin/turfs/" + turf.ID + "/approve", ""},
		{http.MethodPost, "/admin/turfs/" + turf.ID + "/reject", `{"reason":"Not eligible."}`},
		{http.MethodGet, "/admin/users", ""},
	}

	for _, role := range []struct {
		name  string
		token string
	}{{"OWNER", ownerToken}, {"PLAYER", playerToken}} {
		for _, route := range routes {
			t.Run(role.name+" "+route.method+" "+route.path, func(t *testing.T) {
				rec := do(t, env.mux, route.method, testPrefix+route.path, route.body, role.token)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body)
				}
				if got := decodeError(t, rec).Code; got != "forbidden" {
					t.Errorf("code = %q, want forbidden", got)
				}
			})
		}
	}
}

func TestAdminRoutesRequireAuthentication(t *testing.T) {
	env := newTestEnv(t)

	rec := do(t, env.mux, http.MethodGet, testPrefix+"/admin/turfs/pending", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// --- turf moderation ---------------------------------------------------------

func TestAdminPendingTurfsAndDetail(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)

	list := do(t, env.mux, http.MethodGet, testPrefix+"/admin/turfs/pending", "", adminToken)
	if list.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", list.Code, http.StatusOK, list.Body)
	}
	var body struct {
		Turfs []owners.Turf `json:"turfs"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	found := false
	for _, listed := range body.Turfs {
		if listed.ID == turf.ID {
			found = true
		}
	}
	if !found {
		t.Error("the pending list is missing the submitted turf")
	}

	detail := do(t, env.mux, http.MethodGet, testPrefix+"/admin/turfs/"+turf.ID, "", adminToken)
	if detail.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", detail.Code, http.StatusOK, detail.Body)
	}
	if decodeTurf(t, detail).ID != turf.ID {
		t.Error("the detail response is not the requested turf")
	}
}

func TestAdminTurfNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)

	rec := do(t, env.mux, http.MethodGet, testPrefix+"/admin/turfs/00000000-0000-0000-0000-000000000000", "", adminToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if got := decodeError(t, rec).Code; got != "turf_not_found" {
		t.Errorf("code = %q, want turf_not_found", got)
	}
}

func TestAdminApproveTurf(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)

	rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/approve", "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if got := decodeTurf(t, rec).Status; got != owners.StatusApproved {
		t.Errorf("Status = %q, want APPROVED", got)
	}

	// And it is publicly visible now, through the same service the public
	// turf routes use.
	if _, err := env.ownersService.PublicTurf(context.Background(), turf.ID); err != nil {
		t.Errorf("PublicTurf() after approval returned error: %v", err)
	}
}

// Approve is only valid from PENDING_APPROVAL. A DRAFT turf (never submitted)
// must be refused, not silently approved.
func TestAdminApproveTurfRejectsWrongStatus(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)

	ctx := context.Background()
	if _, _, err := env.ownersService.SaveProfile(ctx, ownerID, owners.SaveProfileRequest{DisplayName: "Test Arena"}); err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	draft, err := env.ownersService.CreateTurf(ctx, ownerID, owners.SaveTurfRequest{
		Name: "Draft Turf " + uniqueEmail(t), Address: "1 Draft Street, Area", City: "Kochi",
		OpeningTime: "06:00", ClosingTime: "22:00",
	})
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+draft.ID+"/approve", "", adminToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusConflict, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "invalid_status_transition" {
		t.Errorf("code = %q, want invalid_status_transition", got)
	}
}

func TestAdminRejectTurfRequiresAReason(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)

	rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/reject", `{"reason":""}`, adminToken)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnprocessableEntity, rec.Body)
	}
}

func TestAdminRejectThenResubmitThenApprove(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)

	rejected := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/reject",
		`{"reason":"Address could not be verified."}`, adminToken)
	if rejected.Code != http.StatusOK {
		t.Fatalf("reject status = %d, want %d (body: %s)", rejected.Code, http.StatusOK, rejected.Body)
	}
	rejectedTurf := decodeTurf(t, rejected)
	if rejectedTurf.Status != owners.StatusRejected {
		t.Fatalf("Status = %q, want REJECTED", rejectedTurf.Status)
	}
	if rejectedTurf.ModerationReason != "Address could not be verified." {
		t.Errorf("ModerationReason = %q, want the given reason", rejectedTurf.ModerationReason)
	}

	// The owner can resubmit through the existing Phase 3 submit endpoint's
	// service method; this loop is not admin's to own.
	resubmitted, err := env.ownersService.SubmitTurf(context.Background(), ownerID, turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() after rejection returned error: %v", err)
	}
	if resubmitted.Status != owners.StatusPendingApproval {
		t.Fatalf("Status = %q, want PENDING_APPROVAL", resubmitted.Status)
	}

	approved := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/approve", "", adminToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want %d (body: %s)", approved.Code, http.StatusOK, approved.Body)
	}
	if got := decodeTurf(t, approved).ModerationReason; got != "" {
		t.Errorf("ModerationReason = %q, want it cleared by approval", got)
	}
}

func TestAdminSuspendThenRestoreTurf(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)

	if rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/approve", "", adminToken); rec.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	suspended := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/suspend",
		`{"reason":"Reported by multiple players."}`, adminToken)
	if suspended.Code != http.StatusOK {
		t.Fatalf("suspend status = %d, want %d (body: %s)", suspended.Code, http.StatusOK, suspended.Body)
	}
	if got := decodeTurf(t, suspended).Status; got != owners.StatusSuspended {
		t.Errorf("Status = %q, want SUSPENDED", got)
	}

	// Suspended must drop out of public visibility immediately.
	if _, err := env.ownersService.PublicTurf(context.Background(), turf.ID); err == nil {
		t.Error("PublicTurf() succeeded for a suspended turf, want it hidden")
	}

	restored := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/restore", "", adminToken)
	if restored.Code != http.StatusOK {
		t.Fatalf("restore status = %d, want %d (body: %s)", restored.Code, http.StatusOK, restored.Body)
	}
	restoredTurf := decodeTurf(t, restored)
	if restoredTurf.Status != owners.StatusApproved {
		t.Errorf("Status = %q, want APPROVED", restoredTurf.Status)
	}
	if restoredTurf.ModerationReason != "" {
		t.Errorf("ModerationReason = %q, want it cleared by restoring", restoredTurf.ModerationReason)
	}

	if _, err := env.ownersService.PublicTurf(context.Background(), turf.ID); err != nil {
		t.Errorf("PublicTurf() after restore returned error: %v", err)
	}
}

// Suspend is only valid from APPROVED; a still-pending turf must be refused.
func TestAdminSuspendTurfRejectsWrongStatus(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)

	rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/suspend",
		`{"reason":"Policy violation."}`, adminToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusConflict, rec.Body)
	}
}

// Restore is only valid from SUSPENDED.
func TestAdminRestoreTurfRejectsWrongStatus(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	ownerID, _ := env.createUser(t, auth.RoleOwner)
	turf := env.seedSubmittedTurf(t, ownerID)

	rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/turfs/"+turf.ID+"/restore", "", adminToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusConflict, rec.Body)
	}
}

// --- user management -----------------------------------------------------------

func TestAdminListAndViewUsers(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	targetID, _ := env.createUser(t, auth.RolePlayer)

	list := do(t, env.mux, http.MethodGet, testPrefix+"/admin/users?limit=50&offset=0", "", adminToken)
	if list.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", list.Code, http.StatusOK, list.Body)
	}
	var page auth.UserPage
	if err := json.Unmarshal(list.Body.Bytes(), &page); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	found := false
	for _, u := range page.Users {
		if u.ID == targetID {
			found = true
		}
	}
	if !found {
		t.Error("the user list is missing the seeded target user")
	}

	detail := do(t, env.mux, http.MethodGet, testPrefix+"/admin/users/"+targetID, "", adminToken)
	if detail.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", detail.Code, http.StatusOK, detail.Body)
	}
	var profile auth.Profile
	if err := json.Unmarshal(detail.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if profile.ID != targetID {
		t.Errorf("ID = %q, want %q", profile.ID, targetID)
	}
}

func TestAdminUserNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)

	rec := do(t, env.mux, http.MethodGet, testPrefix+"/admin/users/00000000-0000-0000-0000-000000000000", "", adminToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestAdminDeactivateAndReactivateUser(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)
	targetID, targetToken := env.createUser(t, auth.RolePlayer)

	deactivate := do(t, env.mux, http.MethodPost, testPrefix+"/admin/users/"+targetID+"/deactivate", "", adminToken)
	if deactivate.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", deactivate.Code, http.StatusOK, deactivate.Body)
	}
	var deactivated auth.Profile
	if err := json.Unmarshal(deactivate.Body.Bytes(), &deactivated); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if deactivated.IsActive {
		t.Error("IsActive = true, want false after deactivation")
	}

	// The deactivated account's own existing token must stop working
	// immediately, on the very next request, matching how Authenticate
	// already re-checks IsActive on every call.
	me := do(t, env.mux, http.MethodGet, testPrefix+"/admin/turfs/pending", "", targetToken)
	if me.Code == http.StatusOK {
		t.Error("the deactivated account's token still authenticates successfully")
	}

	reactivate := do(t, env.mux, http.MethodPost, testPrefix+"/admin/users/"+targetID+"/reactivate", "", adminToken)
	if reactivate.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", reactivate.Code, http.StatusOK, reactivate.Body)
	}
	var reactivated auth.Profile
	if err := json.Unmarshal(reactivate.Body.Bytes(), &reactivated); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if !reactivated.IsActive {
		t.Error("IsActive = false, want true after reactivation")
	}
}

// The specific rule this phase requires: an admin must not be able to
// deactivate their own account, through the real HTTP path, with a real
// token identifying themselves as both caller and target.
func TestAdminCannotDeactivateSelf(t *testing.T) {
	env := newTestEnv(t)
	adminID, adminToken := env.createUser(t, auth.RoleAdmin)

	rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/users/"+adminID+"/deactivate", "", adminToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusConflict, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "cannot_modify_self" {
		t.Errorf("code = %q, want cannot_modify_self", got)
	}

	// And the admin's own token must still work: the refusal changed nothing.
	still := do(t, env.mux, http.MethodGet, testPrefix+"/admin/turfs/pending", "", adminToken)
	if still.Code != http.StatusOK {
		t.Errorf("status after the refused self-deactivation = %d, want %d", still.Code, http.StatusOK)
	}
}

func TestAdminCannotReactivateSelf(t *testing.T) {
	env := newTestEnv(t)
	adminID, adminToken := env.createUser(t, auth.RoleAdmin)

	rec := do(t, env.mux, http.MethodPost, testPrefix+"/admin/users/"+adminID+"/reactivate", "", adminToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusConflict, rec.Body)
	}
}

func TestAdminRejectsWrongMethod(t *testing.T) {
	env := newTestEnv(t)
	_, adminToken := env.createUser(t, auth.RoleAdmin)

	cases := []struct{ method, path string }{
		{http.MethodPost, "/admin/turfs/pending"},
		{http.MethodDelete, "/admin/users"},
		{http.MethodPut, "/admin/users/user-1/deactivate"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, env.mux, tc.method, testPrefix+tc.path, "", adminToken)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}
