package httpx

import (
	"encoding/json"
	"testing"
)

type patch struct {
	Name  Optional[string] `json:"name"`
	Count Optional[int]    `json:"count"`
}

// The three states are the whole reason this type exists. A *string collapses
// the last two, which would make "clear this field" indistinguishable from
// "leave it alone".
func TestOptionalDistinguishesAbsentNullAndValue(t *testing.T) {
	var p patch
	if err := json.Unmarshal([]byte(`{"name":null}`), &p); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	if !p.Name.Set {
		t.Error("Name.Set = false for a present key, want true")
	}
	if !p.Name.Clears() {
		t.Error("Name.Clears() = false for an explicit null, want true")
	}
	if _, ok := p.Name.Get(); ok {
		t.Error("Name.Get() reported a value for a null")
	}

	if p.Count.Set {
		t.Error("Count.Set = true for an absent key, want false")
	}
	if p.Count.Clears() {
		t.Error("Count.Clears() = true for an absent key, want false")
	}
}

func TestOptionalDecodesValues(t *testing.T) {
	var p patch
	if err := json.Unmarshal([]byte(`{"name":"pitch","count":3}`), &p); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	if v, ok := p.Name.Get(); !ok || v != "pitch" {
		t.Errorf("Name.Get() = %q, %t, want pitch, true", v, ok)
	}
	if v, ok := p.Count.Get(); !ok || v != 3 {
		t.Errorf("Count.Get() = %d, %t, want 3, true", v, ok)
	}
	if p.Name.Clears() || p.Count.Clears() {
		t.Error("Clears() = true for a supplied value, want false")
	}
}

// An empty string is a value, not an absence. Conflating the two would make it
// impossible to set a field to "".
func TestOptionalTreatsEmptyStringAsAValue(t *testing.T) {
	var p patch
	if err := json.Unmarshal([]byte(`{"name":""}`), &p); err != nil {
		t.Fatalf("decoding failed: %v", err)
	}

	v, ok := p.Name.Get()
	if !ok || v != "" {
		t.Errorf("Name.Get() = %q, %t, want empty string, true", v, ok)
	}
	if p.Name.Clears() {
		t.Error("Clears() = true for an empty string, want false")
	}
}

func TestOptionalReportsDecodeErrors(t *testing.T) {
	var p patch
	if err := json.Unmarshal([]byte(`{"count":"three"}`), &p); err == nil {
		t.Error("decoding a string into Optional[int] returned nil error, want a type error")
	}
}

func TestOptionalZeroValueIsAbsent(t *testing.T) {
	var o Optional[string]

	if o.Set || o.Null || o.Clears() {
		t.Errorf("zero Optional = %+v, want an absent field", o)
	}
	if v, ok := o.Get(); ok || v != "" {
		t.Errorf("Get() = %q, %t, want zero, false", v, ok)
	}
}
