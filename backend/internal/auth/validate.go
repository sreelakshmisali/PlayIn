package auth

import (
	"net/mail"
	"strings"
	"unicode"

	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

// Input limits. They match the CHECK constraints in migration 000002, so a
// request rejected here would have been rejected by the database anyway; doing
// it in the service turns a constraint violation into a useful field error.
const (
	maxEmailLength = 254
	minNameLength  = 2
	maxNameLength  = 120
	minPasswordLen = 10
	// maxPasswordBytes is bcrypt's input limit. Anything past 72 bytes is
	// ignored by the hash, so it is refused rather than silently truncated.
	maxPasswordBytes = 72
)

// RegisterRequest is the body of POST /auth/register.
// Role is optional and defaults to PLAYER.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     Role   `json:"role"`
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest is the body of POST /auth/refresh and POST /auth/logout.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Normalise trims the request and lowercases the email. It runs before
// validation so a trailing space is not reported as an error, and before the
// write so the stored email always satisfies users_email_lowercase_chk.
func (r *RegisterRequest) Normalise() {
	r.Email = normaliseEmail(r.Email)
	r.FullName = strings.TrimSpace(r.FullName)
	if r.Role == "" {
		r.Role = RolePlayer
	} else {
		r.Role = Role(strings.ToUpper(strings.TrimSpace(string(r.Role))))
	}
}

// Validate reports every problem with the request at once, so a client can fix
// a form in one pass instead of one field per round trip.
func (r RegisterRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	errs = appendIf(errs, validateEmail(r.Email))
	errs = appendIf(errs, validatePassword(r.Password))

	switch n := len([]rune(r.FullName)); {
	case n == 0:
		errs = append(errs, field("full_name", "Full name is required."))
	case n < minNameLength || n > maxNameLength:
		errs = append(errs, field("full_name", "Full name must be between 2 and 120 characters."))
	}

	switch {
	case !r.Role.Valid():
		errs = append(errs, field("role", "Role must be PLAYER or OWNER."))
	case !r.Role.SelfAssignable():
		// ADMIN is a valid role but not one an account can claim for itself.
		errs = append(errs, field("role", "Role must be PLAYER or OWNER."))
	}

	return errs
}

// Normalise trims and lowercases the login email so it matches the stored form.
func (r *LoginRequest) Normalise() { r.Email = normaliseEmail(r.Email) }

// Validate checks that both credentials are present.
//
// It deliberately does not apply the registration password rules: tightening
// them later would otherwise lock out every existing account, and a wrong
// password is answered identically either way.
func (r LoginRequest) Validate() []httpx.FieldError {
	var errs []httpx.FieldError

	errs = appendIf(errs, validateEmail(r.Email))
	if r.Password == "" {
		errs = append(errs, field("password", "Password is required."))
	}

	return errs
}

// Validate checks that a refresh token was supplied.
func (r RefreshRequest) Validate() []httpx.FieldError {
	if strings.TrimSpace(r.RefreshToken) == "" {
		return []httpx.FieldError{field("refresh_token", "Refresh token is required.")}
	}
	return nil
}

func normaliseEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateEmail(email string) *httpx.FieldError {
	switch {
	case email == "":
		return ptr(field("email", "Email is required."))
	case len(email) > maxEmailLength:
		return ptr(field("email", "Email must be 254 characters or fewer."))
	}

	// net/mail accepts display-name forms such as `A <a@b.com>`, which are
	// valid addresses but not valid identifiers. Requiring the parsed address
	// to equal the input rules them out.
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || strings.Count(email, "@") != 1 {
		return ptr(field("email", "Email is not a valid address."))
	}

	// net/mail accepts a bare hostname as the domain. The database CHECK
	// requires a dot, so require one here too and report it as a field error.
	_, domain, _ := strings.Cut(email, "@")
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return ptr(field("email", "Email is not a valid address."))
	}
	return nil
}

// validatePassword enforces length and a mix of character classes. Length does
// most of the work; the class rules stop the obvious "aaaaaaaaaa" case.
func validatePassword(password string) *httpx.FieldError {
	switch {
	case password == "":
		return ptr(field("password", "Password is required."))
	case len([]rune(password)) < minPasswordLen:
		return ptr(field("password", "Password must be at least 10 characters."))
	case len(password) > maxPasswordBytes:
		return ptr(field("password", "Password must be 72 bytes or fewer."))
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return ptr(field("password", "Password must contain at least one letter and one number."))
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
