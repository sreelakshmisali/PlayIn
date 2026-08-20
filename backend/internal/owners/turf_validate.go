package owners

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Input limits. They match the CHECK constraints in migration 000004.
const (
	minTurfName   = 2
	maxTurfName   = 120
	maxTurfDesc   = 2000
	minAddress    = 5
	maxAddress    = 250
	minCity       = 2
	maxCity       = 100
	maxImageURL   = 2048
	maxTurfImages = 12
	minLatitude   = -90.0
	maxLatitude   = 90.0
	minLongitude  = -180.0
	maxLongitude  = 180.0
)

// timePattern mirrors the turfs_opening_time_chk / turfs_closing_time_chk
// constraints: 24-hour HH:MM.
var timePattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// SaveTurfRequest is the body of POST /owners/me/turfs and PUT
// /owners/me/turfs/{turfId}. It is a full representation: a field left out is
// stored as empty or NULL, not left alone.
type SaveTurfRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Address     string   `json:"address"`
	City        string   `json:"city"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Capacity    *int32   `json:"capacity"`
	OpeningTime string   `json:"opening_time"`
	ClosingTime string   `json:"closing_time"`
}

// Normalise trims the request.
func (r *SaveTurfRequest) Normalise() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.Address = strings.TrimSpace(r.Address)
	r.City = strings.TrimSpace(r.City)
	r.OpeningTime = strings.TrimSpace(r.OpeningTime)
	r.ClosingTime = strings.TrimSpace(r.ClosingTime)
}

// Validate reports every problem at once.
func (r SaveTurfRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	errs = appendIf(errs, validateTurfName(r.Name))
	errs = appendIf(errs, validateTurfDescription(r.Description))
	errs = appendIf(errs, validateAddress(r.Address))
	errs = appendIf(errs, validateCity(r.City))
	errs = appendIf(errs, validateCapacity(r.Capacity))
	errs = appendIf(errs, validateOpeningTime(r.OpeningTime))
	errs = appendIf(errs, validateClosingTime(r.ClosingTime))
	errs = append(errs, validateLatLng(r.Latitude, r.Longitude)...)

	return errs
}

func (r SaveTurfRequest) fields() turfFields {
	return turfFields{
		Name:        r.Name,
		Description: r.Description,
		Address:     r.Address,
		City:        r.City,
		Latitude:    r.Latitude,
		Longitude:   r.Longitude,
		Capacity:    r.Capacity,
		OpeningTime: r.OpeningTime,
		ClosingTime: r.ClosingTime,
	}
}

// SetTurfSportRequest is the body of POST /owners/me/turfs/{turfId}/sports.
type SetTurfSportRequest struct {
	SportID string `json:"sport_id"`
}

func (r *SetTurfSportRequest) Normalise() { r.SportID = strings.TrimSpace(r.SportID) }

func (r SetTurfSportRequest) Validate() []httpx.FieldError {
	if r.SportID == "" {
		return []httpx.FieldError{field("sport_id", "Sport is required.")}
	}
	return nil
}

// SetTurfAmenityRequest is the body of POST /owners/me/turfs/{turfId}/amenities.
type SetTurfAmenityRequest struct {
	AmenityID string `json:"amenity_id"`
}

func (r *SetTurfAmenityRequest) Normalise() { r.AmenityID = strings.TrimSpace(r.AmenityID) }

func (r SetTurfAmenityRequest) Validate() []httpx.FieldError {
	if r.AmenityID == "" {
		return []httpx.FieldError{field("amenity_id", "Amenity is required.")}
	}
	return nil
}

// AddTurfImageRequest is the body of POST /owners/me/turfs/{turfId}/images.
type AddTurfImageRequest struct {
	ImageURL string `json:"image_url"`
}

func (r *AddTurfImageRequest) Normalise() { r.ImageURL = strings.TrimSpace(r.ImageURL) }

func (r AddTurfImageRequest) Validate() []httpx.FieldError {
	if r.ImageURL == "" {
		return []httpx.FieldError{field("image_url", "Image URL is required.")}
	}
	if err := validateImageURL(r.ImageURL); err != nil {
		return []httpx.FieldError{*err}
	}
	return nil
}

func validateTurfName(value string) *httpx.FieldError {
	switch n := len([]rune(value)); {
	case n == 0:
		return ptr(field("name", "Turf name is required."))
	case n < minTurfName || n > maxTurfName:
		return ptr(field("name", "Turf name must be between 2 and 120 characters."))
	}
	return nil
}

func validateTurfDescription(value string) *httpx.FieldError {
	if len([]rune(value)) > maxTurfDesc {
		return ptr(field("description", "Description must be 2000 characters or fewer."))
	}
	return nil
}

func validateAddress(value string) *httpx.FieldError {
	if n := len([]rune(value)); n < minAddress || n > maxAddress {
		return ptr(field("address", "Address must be between 5 and 250 characters."))
	}
	return nil
}

func validateCity(value string) *httpx.FieldError {
	if n := len([]rune(value)); n < minCity || n > maxCity {
		return ptr(field("city", "City must be between 2 and 100 characters."))
	}
	return nil
}

func validateCapacity(value *int32) *httpx.FieldError {
	if value != nil && *value <= 0 {
		return ptr(field("capacity", "Capacity must be a positive number."))
	}
	return nil
}

func validateOpeningTime(value string) *httpx.FieldError {
	if !timePattern.MatchString(value) {
		return ptr(field("opening_time", "Opening time must be a 24-hour HH:MM value."))
	}
	return nil
}

func validateClosingTime(value string) *httpx.FieldError {
	if !timePattern.MatchString(value) {
		return ptr(field("closing_time", "Closing time must be a 24-hour HH:MM value."))
	}
	return nil
}

// validateLatLng enforces the pairing rule alongside the range: latitude and
// longitude arrive together or not at all, because one without the other is
// not a usable coordinate.
func validateLatLng(lat, lng *float64) []httpx.FieldError {
	var errs []httpx.FieldError

	switch {
	case lat == nil && lng == nil:
		return nil
	case lat == nil:
		errs = append(errs, field("latitude", "Latitude is required when longitude is set."))
	case lng == nil:
		errs = append(errs, field("longitude", "Longitude is required when latitude is set."))
	}

	if lat != nil && (*lat < minLatitude || *lat > maxLatitude) {
		errs = append(errs, field("latitude", "Latitude must be between -90 and 90."))
	}
	if lng != nil && (*lng < minLongitude || *lng > maxLongitude) {
		errs = append(errs, field("longitude", "Longitude must be between -180 and 180."))
	}
	return errs
}

// validateImageURL restricts the scheme to http and https. Anything else,
// javascript: and data: in particular, becomes stored cross-site scripting the
// moment it is rendered into an attribute.
func validateImageURL(value string) *httpx.FieldError {
	if len(value) > maxImageURL {
		return ptr(field("image_url", "Image URL must be 2048 characters or fewer."))
	}

	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ptr(field("image_url", "Image URL must be a full http or https address."))
	}
	return nil
}
