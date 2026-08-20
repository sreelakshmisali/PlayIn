package owners

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

const testPrefix = "/api/v1"

// newTestMux mounts the handler exactly as the router does, so these tests
// exercise the real route table including the guards.
func newTestMux(role auth.Role) (*http.ServeMux, *memStore) {
	handler, store := newTestHandler(testUser(role))

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpx.NotFound)
	handler.Routes(mux, testPrefix)

	return mux, store
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		req.Header.Set("Authorization", "Bearer any-token")
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeProfile(t *testing.T, rec *httptest.ResponseRecorder) Profile {
	t.Helper()
	var profile Profile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decoding the profile failed: %v (body: %s)", err, rec.Body)
	}
	return profile
}

func decodeTurf(t *testing.T, rec *httptest.ResponseRecorder) Turf {
	t.Helper()
	var turf Turf
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

const profileBody = `{"display_name":"Kochi Sports Arena","phone":"+91 98765 43210","description":"Multi-sport turf."}`
const turfBody = `{"name":"Riverside Turf","description":"Nice turf.","address":"123 River Road","city":"Kochi","latitude":9.9312,"longitude":76.2673,"capacity":22,"opening_time":"06:00","closing_time":"22:00"}`

func seedProfileViaAPI(t *testing.T, mux *http.ServeMux) Profile {
	t.Helper()
	rec := do(t, mux, http.MethodPut, testPrefix+"/owners/me", profileBody, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed profile status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
	}
	return decodeProfile(t, rec)
}

func seedTurfViaAPI(t *testing.T, mux *http.ServeMux) Turf {
	t.Helper()
	seedProfileViaAPI(t, mux)

	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs", turfBody, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed turf status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
	}
	return decodeTurf(t, rec)
}

// --- owner profile -----------------------------------------------------------

func TestHandlerOwnerProfileSaveAndFetch(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	seedProfileViaAPI(t, mux)

	rec := do(t, mux, http.MethodGet, testPrefix+"/owners/me", "", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if decodeProfile(t, rec).DisplayName != "Kochi Sports Arena" {
		t.Error("the response is missing the saved display name")
	}
}

func TestHandlerOwnerProfilePatch(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	seedProfileViaAPI(t, mux)

	rec := do(t, mux, http.MethodPatch, testPrefix+"/owners/me", `{"phone":"+91 90000 00000"}`, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	profile := decodeProfile(t, rec)
	if profile.Phone != "+91 90000 00000" {
		t.Errorf("Phone = %q, want the new number", profile.Phone)
	}
	if profile.DisplayName != "Kochi Sports Arena" {
		t.Errorf("DisplayName = %q, want it untouched", profile.DisplayName)
	}
}

// --- turf CRUD -----------------------------------------------------------------

func TestHandlerCreateTurfRequiresOwnerProfileFirst(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)

	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs", turfBody, true)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "owner_profile_not_found" {
		t.Errorf("code = %q, want owner_profile_not_found", got)
	}
}

func TestHandlerCreateAndListTurfs(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	if turf.Status != StatusDraft {
		t.Errorf("Status = %q, want DRAFT", turf.Status)
	}

	rec := do(t, mux, http.MethodGet, testPrefix+"/owners/me/turfs", "", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var body struct {
		Turfs []Turf `json:"turfs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if len(body.Turfs) != 1 {
		t.Fatalf("turfs = %d, want 1", len(body.Turfs))
	}
}

func TestHandlerUpdateAndDeleteTurf(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	renamed := strings.Replace(turfBody, "Riverside Turf", "Renamed Turf", 1)
	rec := do(t, mux, http.MethodPut, testPrefix+"/owners/me/turfs/"+turf.ID, renamed, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if decodeTurf(t, rec).Name != "Renamed Turf" {
		t.Error("the update did not take effect")
	}

	del := do(t, mux, http.MethodDelete, testPrefix+"/owners/me/turfs/"+turf.ID, "", true)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", del.Code, http.StatusNoContent)
	}

	after := do(t, mux, http.MethodGet, testPrefix+"/owners/me/turfs/"+turf.ID, "", true)
	if after.Code != http.StatusNotFound {
		t.Errorf("status after delete = %d, want %d", after.Code, http.StatusNotFound)
	}
}

func TestHandlerCreateTurfValidationErrors(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	seedProfileViaAPI(t, mux)

	body := `{"name":"","address":"","city":"","opening_time":"","closing_time":""}`
	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs", body, true)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnprocessableEntity, rec.Body)
	}
	errBody := decodeError(t, rec)
	if got := fieldNames(errBody.Details); got != "address,city,closing_time,name,opening_time" {
		t.Errorf("fields = %q, want address,city,closing_time,name,opening_time", got)
	}
}

func TestHandlerCreateTurfRejectsDuplicateName(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	seedTurfViaAPI(t, mux)

	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs", turfBody, true)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusConflict, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "turf_name_taken" {
		t.Errorf("code = %q, want turf_name_taken", got)
	}
}

// --- submit --------------------------------------------------------------------

func TestHandlerSubmitTurf(t *testing.T) {
	mux, store := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/submit", "", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if decodeTurf(t, rec).Status != StatusPendingApproval {
		t.Error("the turf did not move to PENDING_APPROVAL")
	}

	// It is now invisible publicly until an admin approves it (Phase 4), and
	// this phase provides no way to reach that state, so PENDING is verified
	// directly against the store.
	store.setStatus(turf.ID, StatusPendingApproval)
}

func TestHandlerSubmitTurfRejectsWrongStatus(t *testing.T) {
	mux, store := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)
	store.setStatus(turf.ID, StatusApproved)

	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/submit", "", true)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusConflict, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "invalid_status_transition" {
		t.Errorf("code = %q, want invalid_status_transition", got)
	}
}

// --- sports, amenities, images --------------------------------------------------

func TestHandlerTurfSports(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	added := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/sports", `{"sport_id":"`+footballID+`"}`, true)
	if added.Code != http.StatusOK {
		t.Fatalf("add status = %d, want %d (body: %s)", added.Code, http.StatusOK, added.Body)
	}
	if len(decodeTurf(t, added).Sports) != 1 {
		t.Fatal("the sport was not attached")
	}

	removed := do(t, mux, http.MethodDelete, testPrefix+"/owners/me/turfs/"+turf.ID+"/sports/"+footballID, "", true)
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want %d (body: %s)", removed.Code, http.StatusOK, removed.Body)
	}
	if len(decodeTurf(t, removed).Sports) != 0 {
		t.Error("the sport is still attached after removal")
	}
}

func TestHandlerTurfAmenities(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	added := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/amenities", `{"amenity_id":"`+parkingID+`"}`, true)
	if added.Code != http.StatusOK {
		t.Fatalf("add status = %d, want %d (body: %s)", added.Code, http.StatusOK, added.Body)
	}
	if len(decodeTurf(t, added).Amenities) != 1 {
		t.Fatal("the amenity was not attached")
	}

	removed := do(t, mux, http.MethodDelete, testPrefix+"/owners/me/turfs/"+turf.ID+"/amenities/"+parkingID, "", true)
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want %d (body: %s)", removed.Code, http.StatusOK, removed.Body)
	}
	if len(decodeTurf(t, removed).Amenities) != 0 {
		t.Error("the amenity is still attached after removal")
	}
}

func TestHandlerTurfImages(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	added := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/images",
		`{"image_url":"https://cdn.playhub.test/1.jpg"}`, true)
	if added.Code != http.StatusOK {
		t.Fatalf("add status = %d, want %d (body: %s)", added.Code, http.StatusOK, added.Body)
	}

	images := decodeTurf(t, added).Images
	if len(images) != 1 {
		t.Fatalf("Images = %+v, want 1 entry", images)
	}

	removed := do(t, mux, http.MethodDelete, testPrefix+"/owners/me/turfs/"+turf.ID+"/images/"+images[0].ID, "", true)
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want %d (body: %s)", removed.Code, http.StatusOK, removed.Body)
	}
	if len(decodeTurf(t, removed).Images) != 0 {
		t.Error("the image is still attached after removal")
	}
}

func TestHandlerTurfSportAndAmenityNotFoundErrors(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"unknown sport", http.MethodPost, "/owners/me/turfs/" + turf.ID + "/sports", `{"sport_id":"sport-nope"}`, http.StatusNotFound, "sport_not_found"},
		{"remove sport not attached", http.MethodDelete, "/owners/me/turfs/" + turf.ID + "/sports/" + footballID, "", http.StatusNotFound, "turf_sport_not_found"},
		{"unknown amenity", http.MethodPost, "/owners/me/turfs/" + turf.ID + "/amenities", `{"amenity_id":"amenity-nope"}`, http.StatusNotFound, "amenity_not_found"},
		{"remove amenity not attached", http.MethodDelete, "/owners/me/turfs/" + turf.ID + "/amenities/" + parkingID, "", http.StatusNotFound, "turf_amenity_not_found"},
		{"remove image not attached", http.MethodDelete, "/owners/me/turfs/" + turf.ID + "/images/image-nope", "", http.StatusNotFound, "turf_image_not_found"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, mux, tc.method, testPrefix+tc.path, tc.body, true)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body)
			}
			if got := decodeError(t, rec).Code; got != tc.wantCode {
				t.Errorf("code = %q, want %q", got, tc.wantCode)
			}
		})
	}
}

func TestHandlerAddTurfImageEnforcesCap(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	for i := 0; i < maxTurfImages; i++ {
		body := `{"image_url":"https://cdn.playhub.test/` + string(rune('a'+i)) + `.jpg"}`
		rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/images", body, true)
		if rec.Code != http.StatusOK {
			t.Fatalf("add image #%d status = %d, want %d (body: %s)", i, rec.Code, http.StatusOK, rec.Body)
		}
	}

	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/images",
		`{"image_url":"https://cdn.playhub.test/over.jpg"}`, true)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnprocessableEntity, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "too_many_images" {
		t.Errorf("code = %q, want too_many_images", got)
	}
}

// Anything unrecognised is a server fault: logged and answered with a generic
// 500 rather than leaking an internal detail to the client.
func TestHandlerUnexpectedStoreFailureIsA500(t *testing.T) {
	mux, store := newTestMux(auth.RoleOwner)
	seedProfileViaAPI(t, mux)
	store.failWith = errors.New("database is on fire")

	rec := do(t, mux, http.MethodGet, testPrefix+"/owners/me", "", true)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body)
	}
	if got := decodeError(t, rec).Code; got != "internal_error" {
		t.Errorf("code = %q, want internal_error", got)
	}
	if strings.Contains(rec.Body.String(), "database is on fire") {
		t.Error("the response leaks the underlying error message")
	}
}

func TestHandlerAddTurfImageRejectsScriptURL(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	rec := do(t, mux, http.MethodPost, testPrefix+"/owners/me/turfs/"+turf.ID+"/images",
		`{"image_url":"javascript:alert(1)"}`, true)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnprocessableEntity, rec.Body)
	}
}

// --- public listing and detail --------------------------------------------------

// This is the core visibility guarantee: DRAFT, PENDING_APPROVAL, REJECTED and
// SUSPENDED turfs must never appear in the public listing or be reachable by
// direct id, and only APPROVED must appear in either.
func TestHandlerPublicVisibilityByStatus(t *testing.T) {
	mux, store := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)

	hidden := []Status{StatusDraft, StatusPendingApproval, StatusRejected, StatusSuspended}
	for _, status := range hidden {
		t.Run(string(status)+" is hidden", func(t *testing.T) {
			store.setStatus(turf.ID, status)

			list := do(t, mux, http.MethodGet, testPrefix+"/turfs", "", false)
			var body struct {
				Turfs []Turf `json:"turfs"`
			}
			if err := json.Unmarshal(list.Body.Bytes(), &body); err != nil {
				t.Fatalf("decoding failed: %v", err)
			}
			if len(body.Turfs) != 0 {
				t.Errorf("public list with status %s returned %d turfs, want 0", status, len(body.Turfs))
			}

			detail := do(t, mux, http.MethodGet, testPrefix+"/turfs/"+turf.ID, "", false)
			if detail.Code != http.StatusNotFound {
				t.Errorf("public detail with status %s = %d, want %d", status, detail.Code, http.StatusNotFound)
			}
		})
	}

	store.setStatus(turf.ID, StatusApproved)

	list := do(t, mux, http.MethodGet, testPrefix+"/turfs", "", false)
	var body struct {
		Turfs []Turf `json:"turfs"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if len(body.Turfs) != 1 {
		t.Fatalf("public list once approved = %d turfs, want 1", len(body.Turfs))
	}

	detail := do(t, mux, http.MethodGet, testPrefix+"/turfs/"+turf.ID, "", false)
	if detail.Code != http.StatusOK {
		t.Fatalf("public detail once approved = %d, want %d (body: %s)", detail.Code, http.StatusOK, detail.Body)
	}
}

func TestHandlerPublicTurfDoesNotExposeOwnerAccountData(t *testing.T) {
	mux, store := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, mux)
	store.setStatus(turf.ID, StatusApproved)

	rec := do(t, mux, http.MethodGet, testPrefix+"/turfs/"+turf.ID, "", false)
	body := rec.Body.String()

	// Quoted as JSON keys or values, so a legitimate field like
	// owner_display_name (which contains "owner") cannot false-positive.
	forbidden := []string{
		`"email"`, "owner@playhub.test", `"password`, `"role"`, `:"OWNER"`,
		`"is_active"`, `"access_token"`, `"refresh_token"`,
	}
	for _, needle := range forbidden {
		if strings.Contains(strings.ToLower(body), strings.ToLower(needle)) {
			t.Errorf("public turf response contains %q, which belongs to the account not the listing:\n%s", needle, body)
		}
	}
	// The owner's business name is legitimate and expected.
	if !strings.Contains(body, "Kochi Sports Arena") {
		t.Error("public turf response is missing the owner display name")
	}
}

func TestHandlerAmenitiesIsPublic(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)

	rec := do(t, mux, http.MethodGet, testPrefix+"/amenities", "", false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var body struct {
		Amenities []Amenity `json:"amenities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if len(body.Amenities) != 2 {
		t.Fatalf("amenities = %d, want 2 in the fixture", len(body.Amenities))
	}
}

// --- ownership isolation across two owner accounts ------------------------------

// The second owner authenticates as a different user id against the same
// handler instance, so this exercises the guard exactly as two real accounts
// would hit it.
func newSecondOwnerMux(store *memStore, userID string) *http.ServeMux {
	svc := NewService(store)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(svc, stubAuthenticator{user: auth.User{ID: userID, Role: auth.RoleOwner, IsActive: true}}, logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/", httpx.NotFound)
	handler.Routes(mux, testPrefix)
	return mux
}

func TestHandlerOneOwnerCannotModifyAnothersTurf(t *testing.T) {
	firstMux, store := newTestMux(auth.RoleOwner)
	turf := seedTurfViaAPI(t, firstMux)

	secondMux := newSecondOwnerMux(store, "user-2")
	do(t, secondMux, http.MethodPut, testPrefix+"/owners/me", `{"display_name":"Rival Turfs"}`, true)

	renamed := strings.Replace(turfBody, "Riverside Turf", "Hijacked", 1)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"get", http.MethodGet, "/owners/me/turfs/" + turf.ID, ""},
		{"update", http.MethodPut, "/owners/me/turfs/" + turf.ID, renamed},
		{"delete", http.MethodDelete, "/owners/me/turfs/" + turf.ID, ""},
		{"submit", http.MethodPost, "/owners/me/turfs/" + turf.ID + "/submit", ""},
		{"add sport", http.MethodPost, "/owners/me/turfs/" + turf.ID + "/sports", `{"sport_id":"` + footballID + `"}`},
		{"add image", http.MethodPost, "/owners/me/turfs/" + turf.ID + "/images", `{"image_url":"https://cdn.playhub.test/x.jpg"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, secondMux, tc.method, testPrefix+tc.path, tc.body, true)
			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body)
			}
		})
	}

	// The turf must be completely untouched by every rejected attempt.
	stillMine := do(t, firstMux, http.MethodGet, testPrefix+"/owners/me/turfs/"+turf.ID, "", true)
	if got := decodeTurf(t, stillMine).Name; got != "Riverside Turf" {
		t.Errorf("Name = %q, want it unchanged", got)
	}

	// And it must not appear in the second owner's own turf list either.
	secondList := do(t, secondMux, http.MethodGet, testPrefix+"/owners/me/turfs", "", true)
	var body struct {
		Turfs []Turf `json:"turfs"`
	}
	if err := json.Unmarshal(secondList.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if len(body.Turfs) != 0 {
		t.Errorf("the second owner's turf list contains %d turfs, want 0", len(body.Turfs))
	}
}

// --- PLAYER is blocked from every owner and turf-management route ---------------

func TestPlayerCannotManageTurfsOrOwnerProfile(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	routes := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/owners/me", ""},
		{http.MethodPut, "/owners/me", profileBody},
		{http.MethodPatch, "/owners/me", `{"phone":"+91 90000 00000"}`},
		{http.MethodPost, "/owners/me/turfs", turfBody},
		{http.MethodGet, "/owners/me/turfs", ""},
		{http.MethodGet, "/owners/me/turfs/turf-1", ""},
		{http.MethodPut, "/owners/me/turfs/turf-1", turfBody},
		{http.MethodDelete, "/owners/me/turfs/turf-1", ""},
		{http.MethodPost, "/owners/me/turfs/turf-1/submit", ""},
		{http.MethodPost, "/owners/me/turfs/turf-1/sports", `{"sport_id":"` + footballID + `"}`},
		{http.MethodPost, "/owners/me/turfs/turf-1/amenities", `{"amenity_id":"` + parkingID + `"}`},
		{http.MethodPost, "/owners/me/turfs/turf-1/images", `{"image_url":"https://cdn.playhub.test/1.jpg"}`},
	}

	for _, tc := range routes {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, mux, tc.method, testPrefix+tc.path, tc.body, true)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body)
			}
			if got := decodeError(t, rec).Code; got != "forbidden" {
				t.Errorf("code = %q, want forbidden", got)
			}
		})
	}
}

// A PLAYER token must not create anything as a side effect of being refused.
func TestPlayerRefusalWritesNothing(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)

	do(t, mux, http.MethodPut, testPrefix+"/owners/me", profileBody, true)

	if len(store.profiles) != 0 {
		t.Errorf("stored owner profiles = %d, want none created for a refused role", len(store.profiles))
	}
}

func TestPlayerCanStillBrowsePublicTurfs(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	if rec := do(t, mux, http.MethodGet, testPrefix+"/turfs", "", true); rec.Code != http.StatusOK {
		t.Errorf("GET /turfs as PLAYER = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec := do(t, mux, http.MethodGet, testPrefix+"/amenities", "", true); rec.Code != http.StatusOK {
		t.Errorf("GET /amenities as PLAYER = %d, want %d", rec.Code, http.StatusOK)
	}
}

// --- unauthenticated -------------------------------------------------------------

func TestOwnerRoutesRequireAuthentication(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)

	paths := []struct{ method, path string }{
		{http.MethodGet, "/owners/me"},
		{http.MethodPut, "/owners/me"},
		{http.MethodPost, "/owners/me/turfs"},
		{http.MethodGet, "/owners/me/turfs"},
	}

	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, mux, tc.method, testPrefix+tc.path, "", false)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestHandlerRejectsWrongMethod(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)

	cases := []struct{ method, path string }{
		{http.MethodDelete, "/owners/me"},
		{http.MethodPatch, "/owners/me/turfs"},
		{http.MethodPost, "/owners/me/turfs/turf-1"},
		{http.MethodGet, "/owners/me/turfs/turf-1/submit"},
		{http.MethodPost, "/turfs"},
		{http.MethodDelete, "/turfs/turf-1"},
		{http.MethodPost, "/amenities"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, mux, tc.method, testPrefix+tc.path, "", true)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHandlerRejectsBadBody(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)
	seedProfileViaAPI(t, mux)

	cases := []struct{ method, path, body string }{
		{http.MethodPut, "/owners/me", `{"display_name":`},
		{http.MethodPost, "/owners/me/turfs", `{"name":"Turf","is_admin":true}`},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, mux, tc.method, testPrefix+tc.path, tc.body, true)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
			}
		})
	}
}
