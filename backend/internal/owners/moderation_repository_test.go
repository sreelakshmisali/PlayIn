package owners

import (
	"context"
	"errors"
	"testing"
)

// createTestAdmin inserts an ADMIN account for the moderated_by audit column
// and removes it when the test ends.
func createTestAdmin(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	var userID string
	err := testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, '$2a$04$notarealhashbutlongenough', 'Test Admin', 'ADMIN')
		RETURNING id::text`, uniqueEmail(t)).Scan(&userID)
	if err != nil {
		t.Fatalf("creating the test admin failed: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID); err != nil {
			t.Errorf("cleaning up admin %s failed: %v", userID, err)
		}
	})
	return userID
}

func submitTestTurf(t *testing.T, repo *Repository, ownerProfileID string) Turf {
	t.Helper()
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	submitted, err := repo.SubmitTurf(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}
	return submitted
}

func TestRepositoryPendingTurfs(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	draft, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	f := testTurfFields()
	f.Name = "Second Turf"
	second, err := repo.CreateTurf(ctx, ownerProfileID, f)
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}
	if _, err := repo.SubmitTurf(ctx, ownerProfileID, second.ID); err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}

	pending, err := repo.PendingTurfs(ctx)
	if err != nil {
		t.Fatalf("PendingTurfs() returned error: %v", err)
	}

	foundDraft, foundSecond := false, false
	for _, turf := range pending {
		if turf.ID == draft.ID {
			foundDraft = true
		}
		if turf.ID == second.ID {
			foundSecond = true
		}
	}
	if foundDraft {
		t.Error("PendingTurfs() includes a DRAFT turf, want only PENDING_APPROVAL")
	}
	if !foundSecond {
		t.Error("PendingTurfs() is missing the submitted turf")
	}
}

func TestRepositoryTurfByIDIsNotOwnerScoped(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	// No owner id argument at all: TurfByID must find it regardless.
	got, err := repo.TurfByID(ctx, turf.ID)
	if err != nil {
		t.Fatalf("TurfByID() returned error: %v", err)
	}
	if got.ID != turf.ID {
		t.Errorf("ID = %q, want %q", got.ID, turf.ID)
	}

	if _, err := repo.TurfByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("TurfByID(unknown) error = %v, want ErrTurfNotFound", err)
	}
	if _, err := repo.TurfByID(ctx, "not-a-uuid"); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("TurfByID(non-uuid) error = %v, want ErrTurfNotFound", err)
	}
}

func TestRepositoryApproveTurf(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	turf := submitTestTurf(t, repo, ownerProfileID)

	approved, err := repo.ApproveTurf(ctx, turf.ID, adminID)
	if err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Errorf("Status = %q, want APPROVED", approved.Status)
	}

	var moderatedBy string
	if err := testPool.QueryRow(ctx, `SELECT moderated_by::text FROM turfs WHERE id = $1`, turf.ID).Scan(&moderatedBy); err != nil {
		t.Fatalf("reading moderated_by failed: %v", err)
	}
	if moderatedBy != adminID {
		t.Errorf("moderated_by = %q, want %q", moderatedBy, adminID)
	}
}

func TestRepositoryApproveTurfRejectsWrongStatus(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	// A fresh turf is DRAFT, not PENDING_APPROVAL.
	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if _, err := repo.ApproveTurf(ctx, turf.ID, adminID); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("ApproveTurf() error = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestRepositoryApproveTurfNotFound(t *testing.T) {
	repo := newTestRepository(t)
	adminID := createTestAdmin(t)

	if _, err := repo.ApproveTurf(context.Background(), "00000000-0000-0000-0000-000000000000", adminID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("ApproveTurf() error = %v, want ErrTurfNotFound", err)
	}
}

func TestRepositoryRejectTurfStoresReason(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	turf := submitTestTurf(t, repo, ownerProfileID)

	rejected, err := repo.RejectTurf(ctx, turf.ID, adminID, "Address does not match the map location.")
	if err != nil {
		t.Fatalf("RejectTurf() returned error: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Errorf("Status = %q, want REJECTED", rejected.Status)
	}
	if rejected.ModerationReason != "Address does not match the map location." {
		t.Errorf("ModerationReason = %q, want the given reason", rejected.ModerationReason)
	}
}

// The CHECK constraint added in 000005 ties a non-null reason to REJECTED or
// SUSPENDED. Exercised directly, since the repository's own queries always
// satisfy it and would never demonstrate it failing on their own.
func TestRepositoryModerationReasonCheckConstraint(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	ctx := context.Background()

	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	_, err = testPool.Exec(ctx, `UPDATE turfs SET moderation_reason = 'Because.' WHERE id = $1`, turf.ID)
	if err == nil {
		t.Error("setting a reason on a DRAFT turf succeeded, want the CHECK constraint to refuse it")
	}
}

func TestRepositorySuspendThenRestoreTurf(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	turf := submitTestTurf(t, repo, ownerProfileID)
	if _, err := repo.ApproveTurf(ctx, turf.ID, adminID); err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}

	suspended, err := repo.SuspendTurf(ctx, turf.ID, adminID, "Reported by multiple players.")
	if err != nil {
		t.Fatalf("SuspendTurf() returned error: %v", err)
	}
	if suspended.Status != StatusSuspended {
		t.Errorf("Status = %q, want SUSPENDED", suspended.Status)
	}

	// Suspended must not be publicly reachable.
	if _, err := repo.PublicTurfByID(ctx, turf.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("PublicTurfByID() on a suspended turf error = %v, want ErrTurfNotFound", err)
	}

	restored, err := repo.RestoreTurf(ctx, turf.ID, adminID)
	if err != nil {
		t.Fatalf("RestoreTurf() returned error: %v", err)
	}
	if restored.Status != StatusApproved {
		t.Errorf("Status = %q, want APPROVED", restored.Status)
	}
	if restored.ModerationReason != "" {
		t.Errorf("ModerationReason = %q, want it cleared", restored.ModerationReason)
	}

	if _, err := repo.PublicTurfByID(ctx, turf.ID); err != nil {
		t.Errorf("PublicTurfByID() after restore returned error: %v", err)
	}
}

func TestRepositorySuspendTurfRejectsWrongStatus(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	// Still DRAFT, never approved.
	turf, err := repo.CreateTurf(ctx, ownerProfileID, testTurfFields())
	if err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	if _, err := repo.SuspendTurf(ctx, turf.ID, adminID, "Policy violation."); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("SuspendTurf() error = %v, want ErrInvalidStatusTransition", err)
	}
}

func TestRepositoryRestoreTurfRejectsWrongStatus(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	turf := submitTestTurf(t, repo, ownerProfileID)

	// PENDING_APPROVAL, never suspended.
	if _, err := repo.RestoreTurf(ctx, turf.ID, adminID); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("RestoreTurf() error = %v, want ErrInvalidStatusTransition", err)
	}
}

// Resubmitting a rejected turf must clear the stale rejection reason, or the
// UPDATE that moves it back to PENDING_APPROVAL would itself violate the
// moderation_reason CHECK constraint (only REJECTED/SUSPENDED may carry one).
func TestRepositoryResubmitAfterRejectionClearsReason(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	turf := submitTestTurf(t, repo, ownerProfileID)
	rejected, err := repo.RejectTurf(ctx, turf.ID, adminID, "Address does not match the map location.")
	if err != nil {
		t.Fatalf("RejectTurf() returned error: %v", err)
	}
	if rejected.ModerationReason == "" {
		t.Fatal("ModerationReason is empty after rejection, want the given reason")
	}

	resubmitted, err := repo.SubmitTurf(ctx, ownerProfileID, turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() after rejection returned error: %v", err)
	}
	if resubmitted.Status != StatusPendingApproval {
		t.Errorf("Status = %q, want PENDING_APPROVAL", resubmitted.Status)
	}
	if resubmitted.ModerationReason != "" {
		t.Errorf("ModerationReason = %q, want it cleared by resubmission", resubmitted.ModerationReason)
	}
}

// moderated_by is ON DELETE SET NULL: deleting the admin account must not
// take the turf, or its moderation history, with it.
func TestRepositoryModeratedByNullsOnAdminDeletion(t *testing.T) {
	repo := newTestRepository(t)
	ownerProfileID := createTestOwnerProfile(t, repo)
	adminID := createTestAdmin(t)
	ctx := context.Background()

	turf := submitTestTurf(t, repo, ownerProfileID)
	if _, err := repo.ApproveTurf(ctx, turf.ID, adminID); err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}

	if _, err := testPool.Exec(ctx, `DELETE FROM users WHERE id = $1`, adminID); err != nil {
		t.Fatalf("deleting the admin failed: %v", err)
	}

	still, err := repo.TurfByID(ctx, turf.ID)
	if err != nil {
		t.Fatalf("TurfByID() after deleting the moderating admin returned error: %v", err)
	}
	if still.Status != StatusApproved {
		t.Errorf("Status = %q, want APPROVED to survive the admin's deletion", still.Status)
	}

	var moderatedBy *string
	if err := testPool.QueryRow(ctx, `SELECT moderated_by::text FROM turfs WHERE id = $1`, turf.ID).Scan(&moderatedBy); err != nil {
		t.Fatalf("reading moderated_by failed: %v", err)
	}
	if moderatedBy != nil {
		t.Errorf("moderated_by = %v, want NULL after the admin was deleted", *moderatedBy)
	}
}
