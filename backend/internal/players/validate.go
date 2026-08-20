package players

import (
	"net/url"
	"strings"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Input limits. They match the CHECK constraints in migration 000003, so a
// request rejected here would have been rejected by the database anyway; doing
// it in the service turns a constraint violation into a useful field error.
const (
	minDisplayName = 2
	maxDisplayName = 80
	maxBio         = 500
	minLocation    = 2
	maxLocation    = 120
	maxImageURL    = 2048
	maxPosition    = 60
)

// SaveProfileRequest is the body of PUT /players/me. It is a full
// representation: a field left out is stored as empty, not left alone. PATCH is
// the verb for leaving things alone.
type SaveProfileRequest struct {
	DisplayName string `json:"display_name"`
	ImageURL    string `json:"image_url"`
	Bio         string `json:"bio"`
	Location    string `json:"location"`
}

// Normalise trims the request. It runs before validation so trailing space is
// not reported as an error, and before the write so stored values satisfy the
// CHECK constraints.
func (r *SaveProfileRequest) Normalise() {
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.ImageURL = strings.TrimSpace(r.ImageURL)
	r.Bio = strings.TrimSpace(r.Bio)
	r.Location = strings.TrimSpace(r.Location)
}

// Validate reports every problem at once, so a client can fix a form in one
// pass instead of one field per round trip.
func (r SaveProfileRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	errs = appendIf(errs, validateDisplayName(r.DisplayName))
	errs = appendIf(errs, validateImageURL(r.ImageURL))
	errs = appendIf(errs, validateBio(r.Bio))
	errs = appendIf(errs, validateLocation(r.Location))

	return errs
}

func (r SaveProfileRequest) fields() profileFields {
	return profileFields{
		DisplayName: r.DisplayName,
		ImageURL:    r.ImageURL,
		Bio:         r.Bio,
		Location:    r.Location,
	}
}

// PatchProfileRequest is the body of PATCH /players/me.
//
// Every field is Optional so the three states a partial update needs stay
// distinct: absent leaves the stored value alone, null clears it, a value
// replaces it. display_name cannot be cleared, because a profile without one
// has nothing to show.
type PatchProfileRequest struct {
	DisplayName httpx.Optional[string] `json:"display_name"`
	ImageURL    httpx.Optional[string] `json:"image_url"`
	Bio         httpx.Optional[string] `json:"bio"`
	Location    httpx.Optional[string] `json:"location"`
}

// Normalise trims every supplied value.
func (r *PatchProfileRequest) Normalise() {
	trim(&r.DisplayName)
	trim(&r.ImageURL)
	trim(&r.Bio)
	trim(&r.Location)
}

// Empty reports whether the body carried no fields at all. A patch that changes
// nothing is a client bug worth naming rather than a successful no-op.
func (r PatchProfileRequest) Empty() bool {
	return !r.DisplayName.Set && !r.ImageURL.Set && !r.Bio.Set && !r.Location.Set
}

// Validate checks only the fields that were supplied.
func (r PatchProfileRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	if r.DisplayName.Clears() {
		errs = append(errs, field("display_name", "Display name cannot be removed."))
	} else if v, ok := r.DisplayName.Get(); ok {
		errs = appendIf(errs, validateDisplayName(v))
	}

	if v, ok := r.ImageURL.Get(); ok {
		errs = appendIf(errs, validateImageURL(v))
	}
	if v, ok := r.Bio.Get(); ok {
		errs = appendIf(errs, validateBio(v))
	}
	if v, ok := r.Location.Get(); ok {
		errs = appendIf(errs, validateLocation(v))
	}

	return errs
}

// apply merges the patch onto the stored values. A field that was not supplied
// keeps what is already there; a field set to null becomes empty, which the
// repository writes as NULL.
func (r PatchProfileRequest) apply(current profileFields) profileFields {
	merged := current

	if v, ok := r.DisplayName.Get(); ok {
		merged.DisplayName = v
	}
	merged.ImageURL = mergeClearable(r.ImageURL, merged.ImageURL)
	merged.Bio = mergeClearable(r.Bio, merged.Bio)
	merged.Location = mergeClearable(r.Location, merged.Location)

	return merged
}

// SetSportRequest is the body of POST /players/me/sports.
type SetSportRequest struct {
	SportID  string `json:"sport_id"`
	Position string `json:"position"`
}

// Normalise trims the request.
func (r *SetSportRequest) Normalise() {
	r.SportID = strings.TrimSpace(r.SportID)
	r.Position = strings.TrimSpace(r.Position)
}

// Validate checks the shape of the request. Whether the sport exists and
// whether it offers the position are storage questions, answered by the write
// itself rather than guessed at here.
func (r SetSportRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	if r.SportID == "" {
		errs = append(errs, field("sport_id", "Sport is required."))
	}
	if len([]rune(r.Position)) > maxPosition {
		errs = append(errs, field("position", "Position must be 60 characters or fewer."))
	}

	return errs
}

func validateDisplayName(value string) *httpx.FieldError {
	switch n := len([]rune(value)); {
	case n == 0:
		return ptr(field("display_name", "Display name is required."))
	case n < minDisplayName || n > maxDisplayName:
		return ptr(field("display_name", "Display name must be between 2 and 80 characters."))
	}
	return nil
}

// validateImageURL accepts an empty string, meaning no image.
//
// The scheme is restricted to http and https. Anything else, javascript: and
// data: in particular, becomes stored cross-site scripting the moment it is
// rendered into an attribute.
func validateImageURL(value string) *httpx.FieldError {
	if value == "" {
		return nil
	}
	if len(value) > maxImageURL {
		return ptr(field("image_url", "Image URL must be 2048 characters or fewer."))
	}

	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ptr(field("image_url", "Image URL must be a full http or https address."))
	}
	return nil
}

func validateBio(value string) *httpx.FieldError {
	if len([]rune(value)) > maxBio {
		return ptr(field("bio", "Bio must be 500 characters or fewer."))
	}
	return nil
}

func validateLocation(value string) *httpx.FieldError {
	if value == "" {
		return nil
	}
	if n := len([]rune(value)); n < minLocation || n > maxLocation {
		return ptr(field("location", "Location must be between 2 and 120 characters."))
	}
	return nil
}

func trim(o *httpx.Optional[string]) {
	if v, ok := o.Get(); ok {
		o.Value = strings.TrimSpace(v)
	}
}

// mergeClearable resolves one optional field against its stored value.
func mergeClearable(patch httpx.Optional[string], current string) string {
	if patch.Clears() {
		return ""
	}
	if v, ok := patch.Get(); ok {
		return v
	}
	return current
}

func field(name, message string) httpx.FieldError {
	return httpx.FieldError{Field: name, Message: message}
}

func appendIf(errs []httpx.FieldError, err *httpx.FieldError) []httpx.FieldError {
	if err == nil {
		return errs
	}
	return append(errs, *err)
}

func ptr[T any](v T) *T { return &v }
