package owners

import (
	"strings"
	"testing"
)

func f64(v float64) *float64 { return &v }
func i32(v int32) *int32     { return &v }

func TestSaveTurfRequestNormalise(t *testing.T) {
	req := SaveTurfRequest{Name: "  Riverside Turf  ", City: "  Kochi  ", OpeningTime: " 06:00 ", ClosingTime: " 22:00 "}
	req.Normalise()

	if req.Name != "Riverside Turf" {
		t.Errorf("Name = %q, want it trimmed", req.Name)
	}
	if req.OpeningTime != "06:00" {
		t.Errorf("OpeningTime = %q, want it trimmed", req.OpeningTime)
	}
}

func TestSaveTurfRequestAccepts(t *testing.T) {
	req := validSaveTurfRequest()
	req.Normalise()

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

// Latitude, longitude and capacity are all optional; the bare minimum is name,
// address, city and the operating hours.
func TestSaveTurfRequestAcceptsBareMinimum(t *testing.T) {
	req := SaveTurfRequest{
		Name: "Riverside Turf", Address: "123 River Road", City: "Kochi",
		OpeningTime: "06:00", ClosingTime: "22:00",
	}
	req.Normalise()

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestSaveTurfRequestRejects(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*SaveTurfRequest)
		wantFields string
	}{
		{"empty name", func(r *SaveTurfRequest) { r.Name = "" }, "name"},
		{"one character name", func(r *SaveTurfRequest) { r.Name = "A" }, "name"},
		{"overlong name", func(r *SaveTurfRequest) { r.Name = strings.Repeat("a", 121) }, "name"},
		{"overlong description", func(r *SaveTurfRequest) { r.Description = strings.Repeat("a", 2001) }, "description"},
		{"short address", func(r *SaveTurfRequest) { r.Address = "123" }, "address"},
		{"overlong address", func(r *SaveTurfRequest) { r.Address = strings.Repeat("a", 251) }, "address"},
		{"short city", func(r *SaveTurfRequest) { r.City = "K" }, "city"},
		{"zero capacity", func(r *SaveTurfRequest) { r.Capacity = i32(0) }, "capacity"},
		{"negative capacity", func(r *SaveTurfRequest) { r.Capacity = i32(-5) }, "capacity"},
		{"bad opening time", func(r *SaveTurfRequest) { r.OpeningTime = "6am" }, "opening_time"},
		{"bad opening hour", func(r *SaveTurfRequest) { r.OpeningTime = "25:00" }, "opening_time"},
		{"bad closing time", func(r *SaveTurfRequest) { r.ClosingTime = "22:75" }, "closing_time"},
		{"latitude out of range", func(r *SaveTurfRequest) { r.Latitude = f64(120) }, "latitude"},
		{"longitude out of range", func(r *SaveTurfRequest) { r.Longitude = f64(-200) }, "longitude"},
		{"latitude without longitude", func(r *SaveTurfRequest) { r.Longitude = nil }, "longitude"},
		{"longitude without latitude", func(r *SaveTurfRequest) { r.Latitude = nil }, "latitude"},
		{"everything wrong", func(r *SaveTurfRequest) {
			r.Name = ""
			r.Address = ""
			r.City = ""
			r.OpeningTime = ""
			r.ClosingTime = ""
			r.Capacity = i32(-1)
			r.Latitude = nil
		}, "address,capacity,city,closing_time,latitude,name,opening_time"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validSaveTurfRequest()
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

func TestSaveTurfRequestAcceptsNoCoordinates(t *testing.T) {
	req := validSaveTurfRequest()
	req.Latitude = nil
	req.Longitude = nil
	req.Normalise()

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestSetTurfSportRequestValidate(t *testing.T) {
	req := SetTurfSportRequest{SportID: "  "}
	req.Normalise()

	if got := fieldNames(req.Validate()); got != "sport_id" {
		t.Errorf("fields = %q, want sport_id", got)
	}

	req = SetTurfSportRequest{SportID: footballID}
	req.Normalise()
	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestSetTurfAmenityRequestValidate(t *testing.T) {
	req := SetTurfAmenityRequest{}
	req.Normalise()

	if got := fieldNames(req.Validate()); got != "amenity_id" {
		t.Errorf("fields = %q, want amenity_id", got)
	}
}

func TestAddTurfImageRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantFields string
	}{
		{"valid", "https://cdn.playhub.test/turf.jpg", ""},
		{"empty", "", "image_url"},
		{"script scheme", "javascript:alert(1)", "image_url"},
		{"no scheme", "cdn.playhub.test/turf.jpg", "image_url"},
		{"overlong", "https://cdn.playhub.test/" + strings.Repeat("a", 2048), "image_url"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := AddTurfImageRequest{ImageURL: tc.url}
			req.Normalise()

			if got := fieldNames(req.Validate()); got != tc.wantFields {
				t.Errorf("fields = %q, want %q", got, tc.wantFields)
			}
		})
	}
}
