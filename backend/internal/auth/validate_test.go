package auth

import (
	"strings"
	"testing"
)

func TestRegisterRequestNormalise(t *testing.T) {
	req := RegisterRequest{
		Email:    "  Player@PlayHub.TEST  ",
		FullName: "  Test Player  ",
		Role:     "owner",
	}
	req.Normalise()

	if req.Email != "player@playhub.test" {
		t.Errorf("Email = %q, want player@playhub.test", req.Email)
	}
	if req.FullName != "Test Player" {
		t.Errorf("FullName = %q, want Test Player", req.FullName)
	}
	if req.Role != RoleOwner {
		t.Errorf("Role = %q, want OWNER", req.Role)
	}
}

func TestRegisterRequestDefaultsToPlayer(t *testing.T) {
	req := RegisterRequest{Email: "a@b.test"}
	req.Normalise()

	if req.Role != RolePlayer {
		t.Errorf("Role = %q, want PLAYER", req.Role)
	}
}

func TestRegisterRequestValidateAccepts(t *testing.T) {
	req := validRegisterRequest()
	req.Normalise()

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestRegisterRequestValidateRejects(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*RegisterRequest)
		wantFields string
	}{
		{"empty email", func(r *RegisterRequest) { r.Email = "" }, "email"},
		{"no at sign", func(r *RegisterRequest) { r.Email = "playhub.test" }, "email"},
		{"no domain dot", func(r *RegisterRequest) { r.Email = "player@localhost" }, "email"},
		{"display name form", func(r *RegisterRequest) { r.Email = "a <a@b.test>" }, "email"},
		{"two at signs", func(r *RegisterRequest) { r.Email = "a@b@c.test" }, "email"},
		{"overlong email", func(r *RegisterRequest) {
			r.Email = strings.Repeat("a", 250) + "@b.test"
		}, "email"},
		{"empty password", func(r *RegisterRequest) { r.Password = "" }, "password"},
		{"short password", func(r *RegisterRequest) { r.Password = "abc12" }, "password"},
		{"letters only", func(r *RegisterRequest) { r.Password = "abcdefghijkl" }, "password"},
		{"digits only", func(r *RegisterRequest) { r.Password = "123456789012" }, "password"},
		{"overlong password", func(r *RegisterRequest) {
			r.Password = strings.Repeat("a1", 40)
		}, "password"},
		{"empty name", func(r *RegisterRequest) { r.FullName = "" }, "full_name"},
		{"one character name", func(r *RegisterRequest) { r.FullName = "A" }, "full_name"},
		{"overlong name", func(r *RegisterRequest) {
			r.FullName = strings.Repeat("A", 121)
		}, "full_name"},
		{"unknown role", func(r *RegisterRequest) { r.Role = "REFEREE" }, "role"},
		// ADMIN is a real role, but not one an account can claim for itself.
		{"self assigned admin", func(r *RegisterRequest) { r.Role = RoleAdmin }, "role"},
		{"everything wrong", func(r *RegisterRequest) {
			r.Email = ""
			r.Password = ""
			r.FullName = ""
			r.Role = "NOPE"
		}, "email,full_name,password,role"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validRegisterRequest()
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

func TestLoginRequestNormaliseAndValidate(t *testing.T) {
	req := LoginRequest{Email: "  Player@PlayHub.TEST ", Password: "anything"}
	req.Normalise()

	if req.Email != "player@playhub.test" {
		t.Errorf("Email = %q, want player@playhub.test", req.Email)
	}
	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

// Login must not apply the registration password rules. Tightening them later
// would otherwise lock out every account created under the old rules.
func TestLoginRequestAcceptsWeakPassword(t *testing.T) {
	req := LoginRequest{Email: "player@playhub.test", Password: "short"}

	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestLoginRequestRejectsMissingFields(t *testing.T) {
	req := LoginRequest{}

	if got := fieldNames(req.Validate()); got != "email,password" {
		t.Errorf("fields = %q, want email,password", got)
	}
}

func TestRefreshRequestValidate(t *testing.T) {
	if problems := (RefreshRequest{RefreshToken: "a.b.c"}).Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
	if got := fieldNames((RefreshRequest{RefreshToken: "   "}).Validate()); got != "refresh_token" {
		t.Errorf("fields = %q, want refresh_token", got)
	}
}

func TestRoleHelpers(t *testing.T) {
	tests := []struct {
		role           Role
		valid          bool
		selfAssignable bool
	}{
		{RolePlayer, true, true},
		{RoleOwner, true, true},
		{RoleAdmin, true, false},
		{Role("REFEREE"), false, false},
		{Role(""), false, false},
	}

	for _, tc := range tests {
		if got := tc.role.Valid(); got != tc.valid {
			t.Errorf("%q.Valid() = %t, want %t", tc.role, got, tc.valid)
		}
		if got := tc.role.SelfAssignable(); got != tc.selfAssignable {
			t.Errorf("%q.SelfAssignable() = %t, want %t", tc.role, got, tc.selfAssignable)
		}
	}
}
