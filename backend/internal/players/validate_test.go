package players

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

func TestSaveProfileRequestNormalise(t *testing.T) {
	req := SaveProfileRequest{
		DisplayName: "  Priya Raman  ",
		ImageURL:    " https://cdn.playhub.test/p.jpg ",
		Bio:         "  Weekend midfielder.  ",
		Location:    "  Kochi  ",
	}
	req.Normalise()

	if req.DisplayName != "Priya Raman" {
		t.Errorf("DisplayName = %q, want it trimmed", req.DisplayName)
	}
	if req.ImageURL != "https://cdn.playhub.test/p.jpg" {
		t.Errorf("ImageURL = %q, want it trimmed", req.ImageURL)
	}
}

func TestSaveProfileRequestAccepts(t *testing.T) {
	req := validSaveRequest()
	req.Normalise()

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

// Everything but the display name is optional.
func TestSaveProfileRequestAcceptsBareMinimum(t *testing.T) {
	req := SaveProfileRequest{DisplayName: "Priya"}
	req.Normalise()

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestSaveProfileRequestRejects(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*SaveProfileRequest)
		wantFields string
	}{
		{"empty name", func(r *SaveProfileRequest) { r.DisplayName = "" }, "display_name"},
		{"blank name", func(r *SaveProfileRequest) { r.DisplayName = "   " }, "display_name"},
		{"one character name", func(r *SaveProfileRequest) { r.DisplayName = "P" }, "display_name"},
		{"overlong name", func(r *SaveProfileRequest) { r.DisplayName = strings.Repeat("a", 81) }, "display_name"},
		{"overlong bio", func(r *SaveProfileRequest) { r.Bio = strings.Repeat("a", 501) }, "bio"},
		{"one character location", func(r *SaveProfileRequest) { r.Location = "K" }, "location"},
		{"overlong location", func(r *SaveProfileRequest) { r.Location = strings.Repeat("a", 121) }, "location"},
		{"scheme relative url", func(r *SaveProfileRequest) { r.ImageURL = "//cdn.playhub.test/p.jpg" }, "image_url"},
		{"bare host", func(r *SaveProfileRequest) { r.ImageURL = "cdn.playhub.test/p.jpg" }, "image_url"},
		{"ftp url", func(r *SaveProfileRequest) { r.ImageURL = "ftp://cdn.playhub.test/p.jpg" }, "image_url"},
		{"overlong url", func(r *SaveProfileRequest) {
			r.ImageURL = "https://cdn.playhub.test/" + strings.Repeat("a", 2048)
		}, "image_url"},
		{"everything wrong", func(r *SaveProfileRequest) {
			r.DisplayName = ""
			r.ImageURL = "javascript:alert(1)"
			r.Bio = strings.Repeat("a", 501)
			r.Location = "K"
		}, "bio,display_name,image_url,location"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validSaveRequest()
			tc.mutate(&req)
			req.Normalise()

			problems := req.Validate()
			if len(problems) == 0 {
				t.Fatal("Validate() returned no problems, want at least one")
			}
			if got := fieldNames(problems); got != tc.wantFields {
				t.Errorf("fields = %q, want %q", got, tc.wantFields)
			}
		})
	}
}

// A javascript: or data: image URL becomes stored cross-site scripting the
// moment it is rendered into an attribute, so the scheme check is not cosmetic.
func TestSaveProfileRequestRejectsScriptURLs(t *testing.T) {
	for _, raw := range []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==",
		"vbscript:msgbox(1)",
	} {
		req := SaveProfileRequest{DisplayName: "Priya", ImageURL: raw}
		req.Normalise()

		if got := fieldNames(req.Validate()); got != "image_url" {
			t.Errorf("Validate(%q) fields = %q, want image_url", raw, got)
		}
	}
}

// The three states a partial update needs must survive JSON decoding, because
// that is the only place the distinction exists.
func TestPatchProfileRequestDecodesThreeStates(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"bio":"hello","location":null}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	if v, ok := req.Bio.Get(); !ok || v != "hello" {
		t.Errorf("Bio.Get() = %q, %t, want hello, true", v, ok)
	}
	if !req.Location.Clears() {
		t.Error("Location.Clears() = false, want true for an explicit null")
	}
	if req.DisplayName.Set {
		t.Error("DisplayName.Set = true, want false for an absent key")
	}
}

func TestPatchProfileRequestEmpty(t *testing.T) {
	var absent PatchProfileRequest
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if !absent.Empty() {
		t.Error("Empty() = false for {}, want true")
	}

	var present PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"bio":null}`), &present); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if present.Empty() {
		t.Error("Empty() = true for an explicit null, want false")
	}
}

// A profile with no display name has nothing to show, so it is the one field a
// patch cannot remove.
func TestPatchProfileRequestRefusesToClearDisplayName(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"display_name":null}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	if got := fieldNames(req.Validate()); got != "display_name" {
		t.Errorf("fields = %q, want display_name", got)
	}
}

func TestPatchProfileRequestValidatesOnlySuppliedFields(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"location":"K"}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	if got := fieldNames(req.Validate()); got != "location" {
		t.Errorf("fields = %q, want location only", got)
	}
}

func TestPatchProfileRequestApply(t *testing.T) {
	current := profileFields{
		DisplayName: "Priya Raman",
		ImageURL:    "https://cdn.playhub.test/p.jpg",
		Bio:         "Weekend midfielder.",
		Location:    "Kochi",
	}

	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"location":"Chennai","bio":null}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	req.Normalise()

	merged := req.apply(current)

	if merged.Location != "Chennai" {
		t.Errorf("Location = %q, want Chennai", merged.Location)
	}
	if merged.Bio != "" {
		t.Errorf("Bio = %q, want it cleared", merged.Bio)
	}
	if merged.DisplayName != current.DisplayName {
		t.Errorf("DisplayName = %q, want it untouched", merged.DisplayName)
	}
	if merged.ImageURL != current.ImageURL {
		t.Errorf("ImageURL = %q, want it untouched", merged.ImageURL)
	}
}

func TestPatchProfileRequestNormaliseTrims(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"display_name":"  Priya  "}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	req.Normalise()

	if v, _ := req.DisplayName.Get(); v != "Priya" {
		t.Errorf("DisplayName = %q, want it trimmed", v)
	}
}

func TestSetSportRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		req        SetSportRequest
		wantFields string
	}{
		{"valid", SetSportRequest{SportID: footballID, Position: "Forward"}, ""},
		{"valid without position", SetSportRequest{SportID: footballID}, ""},
		{"missing sport", SetSportRequest{}, "sport_id"},
		{"blank sport", SetSportRequest{SportID: "   "}, "sport_id"},
		{"overlong position", SetSportRequest{SportID: footballID, Position: strings.Repeat("a", 61)}, "position"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			req.Normalise()

			if got := fieldNames(req.Validate()); got != tc.wantFields {
				t.Errorf("fields = %q, want %q", got, tc.wantFields)
			}
		})
	}
}

func TestOptionalIsUnsetByDefault(t *testing.T) {
	var o httpx.Optional[string]

	if v, ok := o.Get(); ok || v != "" {
		t.Errorf("Get() = %q, %t, want zero, false", v, ok)
	}
	if o.Clears() {
		t.Error("Clears() = true on a zero Optional, want false")
	}
}
