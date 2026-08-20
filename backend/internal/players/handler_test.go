package players

import (
	"encoding/json"
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

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()

	var envelope httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decoding the error body failed: %v (body: %s)", err, rec.Body)
	}
	return envelope.Error
}

const saveBody = `{"display_name":"Priya Raman","image_url":"https://cdn.playhub.test/p.jpg","bio":"Weekend midfielder.","location":"Kochi"}`

// seedViaAPI creates the caller's profile through the handler.
func seedViaAPI(t *testing.T, mux *http.ServeMux) Profile {
	t.Helper()

	rec := do(t, mux, http.MethodPut, testPrefix+"/players/me", saveBody, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
	}
	return decodeProfile(t, rec)
}

func TestHandlerSports(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	// The catalogue is public: no token, still a 200.
	rec := do(t, mux, http.MethodGet, testPrefix+"/sports", "", false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var body struct {
		Sports []Sport `json:"sports"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if len(body.Sports) != 3 {
		t.Fatalf("sports = %d, want 3", len(body.Sports))
	}
	// Positions must reach the client, or it cannot offer the right picker.
	for _, s := range body.Sports {
		if s.Positions == nil {
			t.Errorf("%s positions = nil, want an array", s.Name)
		}
	}
}

func TestHandlerSaveCreatesThenUpdates(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	created := do(t, mux, http.MethodPut, testPrefix+"/players/me", saveBody, true)
	if created.Code != http.StatusCreated {
		t.Fatalf("first save status = %d, want %d (body: %s)", created.Code, http.StatusCreated, created.Body)
	}

	updated := do(t, mux, http.MethodPut, testPrefix+"/players/me", saveBody, true)
	if updated.Code != http.StatusOK {
		t.Errorf("second save status = %d, want %d", updated.Code, http.StatusOK)
	}

	profile := decodeProfile(t, created)
	if profile.DisplayName != "Priya Raman" || profile.Location != "Kochi" {
		t.Errorf("profile = %+v, want the saved values", profile)
	}
}

func TestHandlerMe(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	rec := do(t, mux, http.MethodGet, testPrefix+"/players/me", "", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if decodeProfile(t, rec).UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", decodeProfile(t, rec).UserID)
	}
}

func TestHandlerMeBeforeAProfileExists(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	rec := do(t, mux, http.MethodGet, testPrefix+"/players/me", "", true)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if got := decodeError(t, rec).Code; got != "profile_not_found" {
		t.Errorf("code = %q, want profile_not_found", got)
	}
}

func TestHandlerPatch(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	rec := do(t, mux, http.MethodPatch, testPrefix+"/players/me", `{"location":"Chennai"}`, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	profile := decodeProfile(t, rec)
	if profile.Location != "Chennai" {
		t.Errorf("Location = %q, want Chennai", profile.Location)
	}
	if profile.Bio != "Weekend midfielder." {
		t.Errorf("Bio = %q, want it untouched by the patch", profile.Bio)
	}
}

func TestHandlerPatchRejectsEmptyBody(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	rec := do(t, mux, http.MethodPatch, testPrefix+"/players/me", `{}`, true)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlerValidationErrors(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	body := `{"display_name":"","image_url":"javascript:alert(1)","bio":"","location":"K"}`
	rec := do(t, mux, http.MethodPut, testPrefix+"/players/me", body, true)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnprocessableEntity, rec.Body)
	}

	errBody := decodeError(t, rec)
	if errBody.Code != "validation_failed" {
		t.Errorf("code = %q, want validation_failed", errBody.Code)
	}
	if got := fieldNames(errBody.Details); got != "display_name,image_url,location" {
		t.Errorf("fields = %q, want display_name,image_url,location", got)
	}
}

func TestHandlerAddAndRemoveSport(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	added := do(t, mux, http.MethodPost, testPrefix+"/players/me/sports",
		`{"sport_id":"`+footballID+`","position":"Midfielder"}`, true)
	if added.Code != http.StatusOK {
		t.Fatalf("add status = %d, want %d (body: %s)", added.Code, http.StatusOK, added.Body)
	}

	profile := decodeProfile(t, added)
	if len(profile.Sports) != 1 || profile.Sports[0].Position != "Midfielder" {
		t.Fatalf("sports = %+v, want one Football/Midfielder entry", profile.Sports)
	}

	removed := do(t, mux, http.MethodDelete, testPrefix+"/players/me/sports/"+footballID, "", true)
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want %d (body: %s)", removed.Code, http.StatusOK, removed.Body)
	}
	if len(decodeProfile(t, removed).Sports) != 0 {
		t.Error("the sport is still on the profile after a delete")
	}
}

func TestHandlerAddSportRejections(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"unknown sport", `{"sport_id":"sport-nope"}`, http.StatusNotFound, "sport_not_found"},
		{"retired sport", `{"sport_id":"` + retiredID + `"}`, http.StatusNotFound, "sport_not_found"},
		{"missing sport id", `{"position":"Forward"}`, http.StatusUnprocessableEntity, "validation_failed"},
		{"wrong position", `{"sport_id":"` + footballID + `","position":"Striker"}`, http.StatusUnprocessableEntity, "validation_failed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux, _ := newTestMux(auth.RolePlayer)
			seedViaAPI(t, mux)

			rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/sports", tc.body, true)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body)
			}
			if got := decodeError(t, rec).Code; got != tc.wantCode {
				t.Errorf("code = %q, want %q", got, tc.wantCode)
			}
		})
	}
}

// A rejected position must name the choices, or the client is guessing.
func TestHandlerAddSportPositionErrorListsChoices(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/sports",
		`{"sport_id":"`+footballID+`","position":"Striker"}`, true)

	details := decodeError(t, rec).Details
	if len(details) != 1 || details[0].Field != "position" {
		t.Fatalf("details = %v, want one position error", details)
	}
	if !strings.Contains(details[0].Message, "Goalkeeper") {
		t.Errorf("message = %q, want it to list the valid positions", details[0].Message)
	}
}

func TestHandlerRemoveSportNotPreferred(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	rec := do(t, mux, http.MethodDelete, testPrefix+"/players/me/sports/"+cricketID, "", true)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if got := decodeError(t, rec).Code; got != "sport_not_preferred" {
		t.Errorf("code = %q, want sport_not_preferred", got)
	}
}

func TestHandlerPublicProfile(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	// No token: a profile is meant to be found by other players.
	rec := do(t, mux, http.MethodGet, testPrefix+"/players/user-1", "", false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}
	if decodeProfile(t, rec).DisplayName != "Priya Raman" {
		t.Error("the public profile is missing the display name")
	}
}

func TestHandlerPublicProfileNotFound(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	rec := do(t, mux, http.MethodGet, testPrefix+"/players/user-nobody", "", false)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// The whole point of the separate Profile projection: nothing from the auth
// tables may reach a response, for a stranger or for the owner.
func TestProfileResponsesCarryNoAccountData(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	// Anything that would betray the account behind the profile.
	forbidden := []string{
		"password", "password_hash", "hashed", "$2a$",
		"email", "playhub.test\"", "role", "PLAYER",
		"is_active", "access_token", "refresh_token", "token",
	}

	responses := map[string]*httptest.ResponseRecorder{
		"GET /players/me":   do(t, mux, http.MethodGet, testPrefix+"/players/me", "", true),
		"GET /players/{id}": do(t, mux, http.MethodGet, testPrefix+"/players/user-1", "", false),
		"PUT /players/me":   do(t, mux, http.MethodPut, testPrefix+"/players/me", saveBody, true),
		"PATCH /players/me": do(t, mux, http.MethodPatch, testPrefix+"/players/me", `{"bio":"hi"}`, true),
		"POST /me/sports":   do(t, mux, http.MethodPost, testPrefix+"/players/me/sports", `{"sport_id":"`+footballID+`"}`, true),
	}

	for name, rec := range responses {
		t.Run(name, func(t *testing.T) {
			body := rec.Body.String()
			for _, needle := range forbidden {
				if strings.Contains(strings.ToLower(body), strings.ToLower(needle)) {
					t.Errorf("response contains %q, which belongs to the account not the profile:\n%s", needle, body)
				}
			}
		})
	}
}

// The profile's surrogate key is an internal detail. Exposing it would invite
// clients to address players by it, and it is not the id the rest of the API
// uses.
func TestProfileResponsesHideTheSurrogateKey(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)
	seedViaAPI(t, mux)

	profileID := store.profiles["user-1"].ID

	rec := do(t, mux, http.MethodGet, testPrefix+"/players/user-1", "", false)
	if strings.Contains(rec.Body.String(), profileID) {
		t.Errorf("the response leaks the profile id %q:\n%s", profileID, rec.Body)
	}
}

// Managing a player profile is PLAYER-only. An OWNER or ADMIN token is
// authenticated but must not be handed player functionality.
func TestPlayerRoutesRefuseOtherRoles(t *testing.T) {
	writes := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/players/me", ""},
		{http.MethodPut, "/players/me", saveBody},
		{http.MethodPatch, "/players/me", `{"bio":"hi"}`},
		{http.MethodPost, "/players/me/sports", `{"sport_id":"` + footballID + `"}`},
		{http.MethodDelete, "/players/me/sports/" + footballID, ""},
	}

	for _, role := range []auth.Role{auth.RoleOwner, auth.RoleAdmin} {
		for _, tc := range writes {
			t.Run(string(role)+" "+tc.method+" "+tc.path, func(t *testing.T) {
				mux, _ := newTestMux(role)

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
}

// An ADMIN token must not create a player profile as a side effect of being
// refused, or the refusal is not a refusal.
func TestRefusedRoleWritesNothing(t *testing.T) {
	mux, store := newTestMux(auth.RoleAdmin)

	do(t, mux, http.MethodPut, testPrefix+"/players/me", saveBody, true)

	if len(store.profiles) != 0 {
		t.Errorf("stored profiles = %d, want none created for a refused role", len(store.profiles))
	}
}

func TestPlayerRoutesRequireAuthentication(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	paths := []struct{ method, path string }{
		{http.MethodGet, "/players/me"},
		{http.MethodPut, "/players/me"},
		{http.MethodPatch, "/players/me"},
		{http.MethodPost, "/players/me/sports"},
		{http.MethodDelete, "/players/me/sports/" + footballID},
	}

	for _, tc := range paths {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, mux, tc.method, testPrefix+tc.path, "", false)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", got)
			}
		})
	}
}

func TestHandlerRejectsWrongMethod(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	cases := []struct{ method, path string }{
		{http.MethodDelete, "/players/me"},
		{http.MethodPost, "/players/me"},
		{http.MethodGet, "/players/me/sports"},
		{http.MethodPost, "/sports"},
		{http.MethodDelete, "/players/user-1"},
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
	mux, _ := newTestMux(auth.RolePlayer)

	cases := []struct{ method, path, body string }{
		{http.MethodPut, "/players/me", `{"display_name":`},
		{http.MethodPut, "/players/me", `{"display_name":"Priya","is_admin":true}`},
		{http.MethodPatch, "/players/me", `{"nope":1}`},
		{http.MethodPost, "/players/me/sports", `not json`},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.body, func(t *testing.T) {
			rec := do(t, mux, tc.method, testPrefix+tc.path, tc.body, true)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body)
			}
			if got := decodeError(t, rec).Code; got != "bad_request" {
				t.Errorf("code = %q, want bad_request", got)
			}
		})
	}
}
