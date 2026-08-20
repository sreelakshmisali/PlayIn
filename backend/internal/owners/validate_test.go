package owners

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSaveProfileRequestNormalise(t *testing.T) {
	req := SaveProfileRequest{DisplayName: "  Kochi Sports Arena  ", Phone: " +91 98765 43210 "}
	req.Normalise()

	if req.DisplayName != "Kochi Sports Arena" {
		t.Errorf("DisplayName = %q, want it trimmed", req.DisplayName)
	}
	if req.Phone != "+91 98765 43210" {
		t.Errorf("Phone = %q, want it trimmed", req.Phone)
	}
}

func TestSaveProfileRequestAccepts(t *testing.T) {
	req := validSaveProfileRequest()
	req.Normalise()

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestSaveProfileRequestAcceptsBareMinimum(t *testing.T) {
	req := SaveProfileRequest{DisplayName: "Turf Co"}
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
		{"one character name", func(r *SaveProfileRequest) { r.DisplayName = "A" }, "display_name"},
		{"overlong name", func(r *SaveProfileRequest) { r.DisplayName = strings.Repeat("a", 121) }, "display_name"},
		{"overlong description", func(r *SaveProfileRequest) { r.Description = strings.Repeat("a", 1001) }, "description"},
		{"letters in phone", func(r *SaveProfileRequest) { r.Phone = "call me maybe" }, "phone"},
		{"short phone", func(r *SaveProfileRequest) { r.Phone = "+1234" }, "phone"},
		{"everything wrong", func(r *SaveProfileRequest) {
			r.DisplayName = ""
			r.Phone = "nope"
			r.Description = strings.Repeat("a", 1001)
		}, "description,display_name,phone"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validSaveProfileRequest()
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

func TestPatchProfileRequestDecodesThreeStates(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"phone":"+91 98765 43210","description":null}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	if v, ok := req.Phone.Get(); !ok || v != "+91 98765 43210" {
		t.Errorf("Phone.Get() = %q, %t, want the phone, true", v, ok)
	}
	if !req.Description.Clears() {
		t.Error("Description.Clears() = false for an explicit null, want true")
	}
	if req.DisplayName.Set {
		t.Error("DisplayName.Set = true, want false for an absent key")
	}
}

func TestPatchProfileRequestRefusesToClearDisplayName(t *testing.T) {
	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"display_name":null}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	if got := fieldNames(req.Validate()); got != "display_name" {
		t.Errorf("fields = %q, want display_name", got)
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
	if err := json.Unmarshal([]byte(`{"phone":null}`), &present); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	if present.Empty() {
		t.Error("Empty() = true for an explicit null, want false")
	}
}

func TestPatchProfileRequestApply(t *testing.T) {
	current := profileFields{DisplayName: "Kochi Sports Arena", Phone: "+91 98765 43210", Description: "Nice turf."}

	var req PatchProfileRequest
	if err := json.Unmarshal([]byte(`{"description":null}`), &req); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}
	req.Normalise()

	merged := req.apply(current)

	if merged.Description != "" {
		t.Errorf("Description = %q, want it cleared", merged.Description)
	}
	if merged.Phone != current.Phone {
		t.Errorf("Phone = %q, want it untouched", merged.Phone)
	}
	if merged.DisplayName != current.DisplayName {
		t.Errorf("DisplayName = %q, want it untouched", merged.DisplayName)
	}
}
