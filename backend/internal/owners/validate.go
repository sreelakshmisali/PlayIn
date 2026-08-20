package owners

import (
	"regexp"
	"strings"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// phonePattern mirrors the owner_profiles_phone_chk constraint: a leading
// digit or +, then 6 to 19 more digits, spaces, parentheses or hyphens.
var phonePattern = regexp.MustCompile(`^[+0-9][0-9 ()-]{6,19}$`)

// Input limits. They match the CHECK constraints in migration 000004, so a
// request rejected here would have been rejected by the database anyway; doing
// it in the service turns a constraint violation into a useful field error.
const (
	minDisplayName = 2
	maxDisplayName = 120
	maxDescription = 1000
)

// SaveProfileRequest is the body of PUT /owners/me. It is a full
// representation: a field left out is stored as empty, not left alone.
type SaveProfileRequest struct {
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
}

// Normalise trims the request. It runs before validation so trailing space is
// not reported as an error, and before the write so stored values satisfy the
// CHECK constraints.
func (r *SaveProfileRequest) Normalise() {
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Phone = strings.TrimSpace(r.Phone)
	r.Description = strings.TrimSpace(r.Description)
}

// Validate reports every problem at once, so a client can fix a form in one
// pass instead of one field per round trip.
func (r SaveProfileRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	errs = appendIf(errs, validateDisplayName(r.DisplayName))
	errs = appendIf(errs, validatePhone(r.Phone))
	errs = appendIf(errs, validateDescription(r.Description))

	return errs
}

func (r SaveProfileRequest) fields() profileFields {
	return profileFields{DisplayName: r.DisplayName, Phone: r.Phone, Description: r.Description}
}

// PatchProfileRequest is the body of PATCH /owners/me.
//
// Every field is Optional so a partial update can distinguish leaving a field
// alone from clearing it. DisplayName cannot be cleared, because a profile
// without one has nothing to show.
type PatchProfileRequest struct {
	DisplayName httpx.Optional[string] `json:"display_name"`
	Phone       httpx.Optional[string] `json:"phone"`
	Description httpx.Optional[string] `json:"description"`
}

// Normalise trims every supplied value.
func (r *PatchProfileRequest) Normalise() {
	trim(&r.DisplayName)
	trim(&r.Phone)
	trim(&r.Description)
}

// Empty reports whether the body carried no fields at all.
func (r PatchProfileRequest) Empty() bool {
	return !r.DisplayName.Set && !r.Phone.Set && !r.Description.Set
}

// Validate checks only the fields that were supplied.
func (r PatchProfileRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	if r.DisplayName.Clears() {
		errs = append(errs, field("display_name", "Display name cannot be removed."))
	} else if v, ok := r.DisplayName.Get(); ok {
		errs = appendIf(errs, validateDisplayName(v))
	}

	if v, ok := r.Phone.Get(); ok {
		errs = appendIf(errs, validatePhone(v))
	}
	if v, ok := r.Description.Get(); ok {
		errs = appendIf(errs, validateDescription(v))
	}

	return errs
}

// apply merges the patch onto the stored values.
func (r PatchProfileRequest) apply(current profileFields) profileFields {
	merged := current
	if v, ok := r.DisplayName.Get(); ok {
		merged.DisplayName = v
	}
	merged.Phone = mergeClearable(r.Phone, merged.Phone)
	merged.Description = mergeClearable(r.Description, merged.Description)
	return merged
}

func validateDisplayName(value string) *httpx.FieldError {
	switch n := len([]rune(value)); {
	case n == 0:
		return ptr(field("display_name", "Display name is required."))
	case n < minDisplayName || n > maxDisplayName:
		return ptr(field("display_name", "Display name must be between 2 and 120 characters."))
	}
	return nil
}

// validatePhone accepts an empty string, meaning no phone number given.
// The pattern matches the CHECK constraint: a leading digit or +, then 6 to 19
// more digits, spaces, parentheses or hyphens.
func validatePhone(value string) *httpx.FieldError {
	if value == "" {
		return nil
	}
	if !phonePattern.MatchString(value) {
		return ptr(field("phone", "Phone must be a valid contact number."))
	}
	return nil
}

func validateDescription(value string) *httpx.FieldError {
	if len([]rune(value)) > maxDescription {
		return ptr(field("description", "Description must be 1000 characters or fewer."))
	}
	return nil
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
