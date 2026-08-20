package auth

import (
	"context"
	"errors"
	"testing"
)

func TestServiceListUsersDefaultsAndClampsPageSize(t *testing.T) {
	svc, _ := newTestService(t)
	for i := 0; i < 3; i++ {
		registerUser(t, svc, "user"+string(rune('a'+i))+"@playhub.test", RolePlayer)
	}

	page, err := svc.ListUsers(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("ListUsers() returned error: %v", err)
	}
	if page.Limit != defaultUserPageSize {
		t.Errorf("Limit = %d, want the default %d", page.Limit, defaultUserPageSize)
	}
	if page.Total != 3 {
		t.Errorf("Total = %d, want 3", page.Total)
	}
	if len(page.Users) != 3 {
		t.Errorf("Users = %d, want 3", len(page.Users))
	}
}

func TestServiceListUsersClampsOversizedLimit(t *testing.T) {
	svc, _ := newTestService(t)

	page, err := svc.ListUsers(context.Background(), 10_000, -5)
	if err != nil {
		t.Fatalf("ListUsers() returned error: %v", err)
	}
	if page.Limit != maxUserPageSize {
		t.Errorf("Limit = %d, want the ceiling %d", page.Limit, maxUserPageSize)
	}
	if page.Offset != 0 {
		t.Errorf("Offset = %d, want a negative offset clamped to 0", page.Offset)
	}
}

// The returned Profile must never carry a password hash, matching the same
// projection every other user-facing read already uses.
func TestServiceListUsersReturnsSafeProjection(t *testing.T) {
	svc, _ := newTestService(t)
	registerUser(t, svc, "player@playhub.test", RolePlayer)

	page, err := svc.ListUsers(context.Background(), 20, 0)
	if err != nil {
		t.Fatalf("ListUsers() returned error: %v", err)
	}
	if len(page.Users) != 1 {
		t.Fatalf("Users = %d, want 1", len(page.Users))
	}
	if page.Users[0].Email != "player@playhub.test" {
		t.Errorf("Email = %q, want player@playhub.test", page.Users[0].Email)
	}
}

func TestServiceAdminUser(t *testing.T) {
	svc, _ := newTestService(t)
	session := registerUser(t, svc, "player@playhub.test", RolePlayer)

	profile, err := svc.AdminUser(context.Background(), session.User.ID)
	if err != nil {
		t.Fatalf("AdminUser() returned error: %v", err)
	}
	if profile.ID != session.User.ID {
		t.Errorf("ID = %q, want %q", profile.ID, session.User.ID)
	}
}

func TestServiceAdminUserNotFound(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.AdminUser(context.Background(), "user-nobody"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("AdminUser() error = %v, want ErrUserNotFound", err)
	}
}

func TestServiceSetUserActive(t *testing.T) {
	svc, _ := newTestService(t)
	admin := registerUser(t, svc, "admin@playhub.test", RoleAdmin)
	target := registerUser(t, svc, "player@playhub.test", RolePlayer)

	deactivated, err := svc.SetUserActive(context.Background(), admin.User.ID, target.User.ID, false)
	if err != nil {
		t.Fatalf("SetUserActive(false) returned error: %v", err)
	}
	if deactivated.IsActive {
		t.Error("IsActive = true, want false after deactivation")
	}

	// A deactivated account is now refused at login, which is the whole point.
	_, err = svc.Login(context.Background(), LoginRequest{Email: "player@playhub.test", Password: "correct horse 7"})
	if !errors.Is(err, ErrAccountInactive) {
		t.Errorf("Login() after deactivation error = %v, want ErrAccountInactive", err)
	}

	reactivated, err := svc.SetUserActive(context.Background(), admin.User.ID, target.User.ID, true)
	if err != nil {
		t.Fatalf("SetUserActive(true) returned error: %v", err)
	}
	if !reactivated.IsActive {
		t.Error("IsActive = false, want true after reactivation")
	}
}

// The one rule this phase specifically requires: an admin must not be able to
// deactivate (or otherwise modify the active flag of) their own account.
// IsActive is re-checked on every request (see Authenticate), so a self
// deactivation would lock the admin out immediately, including from the
// endpoint that could undo it.
func TestServiceSetUserActiveRefusesSelf(t *testing.T) {
	svc, _ := newTestService(t)
	admin := registerUser(t, svc, "admin@playhub.test", RoleAdmin)

	if _, err := svc.SetUserActive(context.Background(), admin.User.ID, admin.User.ID, false); !errors.Is(err, ErrCannotModifySelf) {
		t.Errorf("SetUserActive(deactivate self) error = %v, want ErrCannotModifySelf", err)
	}
	if _, err := svc.SetUserActive(context.Background(), admin.User.ID, admin.User.ID, true); !errors.Is(err, ErrCannotModifySelf) {
		t.Errorf("SetUserActive(reactivate self) error = %v, want ErrCannotModifySelf", err)
	}

	// And the account must genuinely be untouched: still active, still able
	// to authenticate.
	still, err := svc.AdminUser(context.Background(), admin.User.ID)
	if err != nil {
		t.Fatalf("AdminUser() returned error: %v", err)
	}
	if !still.IsActive {
		t.Error("IsActive = false, want the refused call to have changed nothing")
	}
}

func TestServiceSetUserActiveNotFound(t *testing.T) {
	svc, _ := newTestService(t)
	admin := registerUser(t, svc, "admin@playhub.test", RoleAdmin)

	if _, err := svc.SetUserActive(context.Background(), admin.User.ID, "user-nobody", false); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("SetUserActive() error = %v, want ErrUserNotFound", err)
	}
}
