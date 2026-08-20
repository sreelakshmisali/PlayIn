package auth

import (
	"context"
	"errors"
	"testing"
)

func TestRepositoryUsersAndUserCount(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	baseline, err := repo.UserCount(ctx)
	if err != nil {
		t.Fatalf("UserCount() returned error: %v", err)
	}

	first := createTestUser(t, repo, RolePlayer)
	second := createTestUser(t, repo, RoleOwner)

	total, err := repo.UserCount(ctx)
	if err != nil {
		t.Fatalf("UserCount() returned error: %v", err)
	}
	if total != baseline+2 {
		t.Errorf("UserCount() = %d, want %d", total, baseline+2)
	}

	// A page big enough to hold everything created here, newest first.
	page, err := repo.Users(ctx, baseline+2, 0)
	if err != nil {
		t.Fatalf("Users() returned error: %v", err)
	}
	if len(page) != baseline+2 {
		t.Fatalf("Users() returned %d rows, want %d", len(page), baseline+2)
	}
	if page[0].ID != second.ID {
		t.Errorf("Users()[0].ID = %q, want the most recently created user %q", page[0].ID, second.ID)
	}

	// No password hash reaches the caller through anything but the one field
	// meant to carry it, and even that is never serialised (see User's own
	// doc comment); this just confirms the row scan is complete.
	found := false
	for _, u := range page {
		if u.ID == first.ID {
			found = true
			if u.PasswordHash == "" {
				t.Error("PasswordHash is empty, want the stored hash")
			}
		}
	}
	if !found {
		t.Error("Users() is missing a user created in this test")
	}
}

func TestRepositoryUsersPagination(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		createTestUser(t, repo, RolePlayer)
	}

	firstPage, err := repo.Users(ctx, 2, 0)
	if err != nil {
		t.Fatalf("Users() returned error: %v", err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("first page = %d users, want 2", len(firstPage))
	}

	secondPage, err := repo.Users(ctx, 2, 2)
	if err != nil {
		t.Fatalf("Users() returned error: %v", err)
	}
	if len(secondPage) == 0 {
		t.Fatal("second page = 0 users, want at least 1")
	}

	// The pages must not overlap.
	for _, a := range firstPage {
		for _, b := range secondPage {
			if a.ID == b.ID {
				t.Errorf("user %s appears on both pages", a.ID)
			}
		}
	}
}

func TestRepositorySetActive(t *testing.T) {
	repo := newTestRepository(t)
	user := createTestUser(t, repo, RolePlayer)
	ctx := context.Background()

	deactivated, err := repo.SetActive(ctx, user.ID, false)
	if err != nil {
		t.Fatalf("SetActive(false) returned error: %v", err)
	}
	if deactivated.IsActive {
		t.Error("IsActive = true, want false")
	}

	reactivated, err := repo.SetActive(ctx, user.ID, true)
	if err != nil {
		t.Fatalf("SetActive(true) returned error: %v", err)
	}
	if !reactivated.IsActive {
		t.Error("IsActive = false, want true")
	}
}

func TestRepositorySetActiveNotFound(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if _, err := repo.SetActive(ctx, "00000000-0000-0000-0000-000000000000", false); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("SetActive(unknown) error = %v, want ErrUserNotFound", err)
	}
	if _, err := repo.SetActive(ctx, "not-a-uuid", false); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("SetActive(non-uuid) error = %v, want ErrUserNotFound", err)
	}
}
