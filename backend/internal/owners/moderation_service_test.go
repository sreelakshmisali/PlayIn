package owners

import (
	"context"
	"errors"
	"testing"
)

const adminUserID = "admin-1"

func TestServicePendingTurfsOnlyListsSubmitted(t *testing.T) {
	svc, store := newTestService()
	ctx := context.Background()

	draft := seedTurf(t, svc, store, "user-1")
	store.setStatus(draft.ID, StatusDraft)

	pending, err := svc.SubmitTurf(ctx, "user-1", draft.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}

	list, err := svc.PendingTurfs(ctx)
	if err != nil {
		t.Fatalf("PendingTurfs() returned error: %v", err)
	}
	if len(list) != 1 || list[0].ID != pending.ID {
		t.Fatalf("PendingTurfs() = %+v, want just the submitted turf", list)
	}
}

func TestServiceAdminTurfIsNotOwnerScoped(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	// AdminTurf takes no owner id at all; it must find the turf regardless of
	// who owns it, unlike every owner-facing lookup in this package.
	got, err := svc.AdminTurf(context.Background(), turf.ID)
	if err != nil {
		t.Fatalf("AdminTurf() returned error: %v", err)
	}
	if got.ID != turf.ID {
		t.Errorf("ID = %q, want %q", got.ID, turf.ID)
	}
}

func TestServiceAdminTurfNotFound(t *testing.T) {
	svc, _ := newTestService()

	if _, err := svc.AdminTurf(context.Background(), "turf-nope"); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("AdminTurf() error = %v, want ErrTurfNotFound", err)
	}
}

// submitted returns a turf already in PENDING_APPROVAL, the only status the
// approve and reject transitions accept.
func submitted(t *testing.T, svc *Service, store *memStore, userID string) Turf {
	t.Helper()
	turf := seedTurf(t, svc, store, userID)
	turf, err := svc.SubmitTurf(context.Background(), userID, turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}
	return turf
}

func TestServiceApproveTurf(t *testing.T) {
	svc, store := newTestService()
	turf := submitted(t, svc, store, "user-1")

	approved, err := svc.ApproveTurf(context.Background(), turf.ID, adminUserID)
	if err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Errorf("Status = %q, want APPROVED", approved.Status)
	}
}

// Approving a turf clears any reason left over from a previous rejection so a
// resubmitted-then-approved turf does not carry a stale explanation.
func TestServiceApproveTurfClearsAStaleReason(t *testing.T) {
	svc, store := newTestService()
	turf := submitted(t, svc, store, "user-1")

	rejected, err := svc.RejectTurf(context.Background(), turf.ID, adminUserID, ModerateTurfRequest{Reason: "Missing photos."})
	if err != nil {
		t.Fatalf("RejectTurf() returned error: %v", err)
	}
	if rejected.ModerationReason == "" {
		t.Fatal("the rejection did not record a reason")
	}

	resubmitted, err := svc.SubmitTurf(context.Background(), "user-1", turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}

	approved, err := svc.ApproveTurf(context.Background(), resubmitted.ID, adminUserID)
	if err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}
	if approved.ModerationReason != "" {
		t.Errorf("ModerationReason = %q, want it cleared by approval", approved.ModerationReason)
	}
}

func TestServiceApproveTurfRejectsWrongStatus(t *testing.T) {
	for _, status := range []Status{StatusDraft, StatusApproved, StatusRejected, StatusSuspended} {
		t.Run(string(status), func(t *testing.T) {
			svc, store := newTestService()
			turf := seedTurf(t, svc, store, "user-1")
			store.setStatus(turf.ID, status)

			if _, err := svc.ApproveTurf(context.Background(), turf.ID, adminUserID); !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("ApproveTurf() error = %v, want ErrInvalidStatusTransition", err)
			}
		})
	}
}

func TestServiceApproveTurfNotFound(t *testing.T) {
	svc, _ := newTestService()

	if _, err := svc.ApproveTurf(context.Background(), "turf-nope", adminUserID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("ApproveTurf() error = %v, want ErrTurfNotFound", err)
	}
}

func TestServiceRejectTurf(t *testing.T) {
	svc, store := newTestService()
	turf := submitted(t, svc, store, "user-1")

	rejected, err := svc.RejectTurf(context.Background(), turf.ID, adminUserID, ModerateTurfRequest{Reason: "Address could not be verified."})
	if err != nil {
		t.Fatalf("RejectTurf() returned error: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Errorf("Status = %q, want REJECTED", rejected.Status)
	}
	if rejected.ModerationReason != "Address could not be verified." {
		t.Errorf("ModerationReason = %q, want the given reason", rejected.ModerationReason)
	}
}

func TestServiceRejectTurfRejectsWrongStatus(t *testing.T) {
	for _, status := range []Status{StatusDraft, StatusApproved, StatusRejected, StatusSuspended} {
		t.Run(string(status), func(t *testing.T) {
			svc, store := newTestService()
			turf := seedTurf(t, svc, store, "user-1")
			store.setStatus(turf.ID, status)

			_, err := svc.RejectTurf(context.Background(), turf.ID, adminUserID, ModerateTurfRequest{Reason: "Not eligible."})
			if !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("RejectTurf() error = %v, want ErrInvalidStatusTransition", err)
			}
		})
	}
}

// A rejected turf can be edited and resubmitted by its owner: this is the
// loop back into the existing Phase 3 submit transition, not a new one.
func TestServiceRejectedTurfCanBeResubmitted(t *testing.T) {
	svc, store := newTestService()
	turf := submitted(t, svc, store, "user-1")

	if _, err := svc.RejectTurf(context.Background(), turf.ID, adminUserID, ModerateTurfRequest{Reason: "Needs more photos."}); err != nil {
		t.Fatalf("RejectTurf() returned error: %v", err)
	}

	resubmitted, err := svc.SubmitTurf(context.Background(), "user-1", turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() after rejection returned error: %v", err)
	}
	if resubmitted.Status != StatusPendingApproval {
		t.Errorf("Status = %q, want PENDING_APPROVAL", resubmitted.Status)
	}
}

func TestServiceSuspendTurf(t *testing.T) {
	svc, store := newTestService()
	turf := submitted(t, svc, store, "user-1")

	approved, err := svc.ApproveTurf(context.Background(), turf.ID, adminUserID)
	if err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}

	suspended, err := svc.SuspendTurf(context.Background(), approved.ID, adminUserID, ModerateTurfRequest{Reason: "Reported for a safety issue."})
	if err != nil {
		t.Fatalf("SuspendTurf() returned error: %v", err)
	}
	if suspended.Status != StatusSuspended {
		t.Errorf("Status = %q, want SUSPENDED", suspended.Status)
	}
	if suspended.ModerationReason != "Reported for a safety issue." {
		t.Errorf("ModerationReason = %q, want the given reason", suspended.ModerationReason)
	}
}

func TestServiceSuspendTurfRejectsWrongStatus(t *testing.T) {
	for _, status := range []Status{StatusDraft, StatusPendingApproval, StatusRejected, StatusSuspended} {
		t.Run(string(status), func(t *testing.T) {
			svc, store := newTestService()
			turf := seedTurf(t, svc, store, "user-1")
			store.setStatus(turf.ID, status)

			_, err := svc.SuspendTurf(context.Background(), turf.ID, adminUserID, ModerateTurfRequest{Reason: "Policy violation."})
			if !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("SuspendTurf() error = %v, want ErrInvalidStatusTransition", err)
			}
		})
	}
}

// A suspended turf must disappear from public visibility immediately: it is
// still not APPROVED, which is the only status the public queries accept.
func TestServiceSuspendedTurfIsNotPubliclyVisible(t *testing.T) {
	svc, store := newTestService()
	turf := submitted(t, svc, store, "user-1")

	approved, err := svc.ApproveTurf(context.Background(), turf.ID, adminUserID)
	if err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}
	if _, err := svc.PublicTurf(context.Background(), approved.ID); err != nil {
		t.Fatalf("PublicTurf() before suspension returned error: %v", err)
	}

	if _, err := svc.SuspendTurf(context.Background(), approved.ID, adminUserID, ModerateTurfRequest{Reason: "Under review."}); err != nil {
		t.Fatalf("SuspendTurf() returned error: %v", err)
	}

	if _, err := svc.PublicTurf(context.Background(), approved.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("PublicTurf() after suspension error = %v, want ErrTurfNotFound", err)
	}
	turfs, err := svc.PublicTurfs(context.Background())
	if err != nil {
		t.Fatalf("PublicTurfs() returned error: %v", err)
	}
	for _, listed := range turfs {
		if listed.ID == approved.ID {
			t.Error("PublicTurfs() includes a suspended turf, want it excluded")
		}
	}
}

func TestServiceRestoreTurf(t *testing.T) {
	svc, store := newTestService()
	turf := submitted(t, svc, store, "user-1")

	approved, err := svc.ApproveTurf(context.Background(), turf.ID, adminUserID)
	if err != nil {
		t.Fatalf("ApproveTurf() returned error: %v", err)
	}
	if _, err := svc.SuspendTurf(context.Background(), approved.ID, adminUserID, ModerateTurfRequest{Reason: "Temporary hold."}); err != nil {
		t.Fatalf("SuspendTurf() returned error: %v", err)
	}

	restored, err := svc.RestoreTurf(context.Background(), approved.ID, adminUserID)
	if err != nil {
		t.Fatalf("RestoreTurf() returned error: %v", err)
	}
	if restored.Status != StatusApproved {
		t.Errorf("Status = %q, want APPROVED", restored.Status)
	}
	if restored.ModerationReason != "" {
		t.Errorf("ModerationReason = %q, want it cleared by restoring", restored.ModerationReason)
	}

	// And it is publicly visible again.
	if _, err := svc.PublicTurf(context.Background(), restored.ID); err != nil {
		t.Errorf("PublicTurf() after restore returned error: %v", err)
	}
}

func TestServiceRestoreTurfRejectsWrongStatus(t *testing.T) {
	for _, status := range []Status{StatusDraft, StatusPendingApproval, StatusApproved, StatusRejected} {
		t.Run(string(status), func(t *testing.T) {
			svc, store := newTestService()
			turf := seedTurf(t, svc, store, "user-1")
			store.setStatus(turf.ID, status)

			if _, err := svc.RestoreTurf(context.Background(), turf.ID, adminUserID); !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("RestoreTurf() error = %v, want ErrInvalidStatusTransition", err)
			}
		})
	}
}

// The full lifecycle in one pass, as a single integration-style check that the
// state machine composes correctly end to end.
func TestServiceFullModerationLifecycle(t *testing.T) {
	svc, store := newTestService()
	ctx := context.Background()

	turf := seedTurf(t, svc, store, "user-1")
	if turf.Status != StatusDraft {
		t.Fatalf("new turf status = %q, want DRAFT", turf.Status)
	}

	firstSubmit, err := svc.SubmitTurf(ctx, "user-1", turf.ID)
	if err != nil || firstSubmit.Status != StatusPendingApproval {
		t.Fatalf("submit: got %+v, err %v", firstSubmit, err)
	}

	rejected, err := svc.RejectTurf(ctx, turf.ID, adminUserID, ModerateTurfRequest{Reason: "Please add opening hours detail."})
	if err != nil || rejected.Status != StatusRejected {
		t.Fatalf("reject: got %+v, err %v", rejected, err)
	}

	resubmitted, err := svc.SubmitTurf(ctx, "user-1", turf.ID)
	if err != nil || resubmitted.Status != StatusPendingApproval {
		t.Fatalf("resubmit: got %+v, err %v", resubmitted, err)
	}

	approved, err := svc.ApproveTurf(ctx, turf.ID, adminUserID)
	if err != nil || approved.Status != StatusApproved {
		t.Fatalf("approve: got %+v, err %v", approved, err)
	}

	suspended, err := svc.SuspendTurf(ctx, turf.ID, adminUserID, ModerateTurfRequest{Reason: "Reported by a player."})
	if err != nil || suspended.Status != StatusSuspended {
		t.Fatalf("suspend: got %+v, err %v", suspended, err)
	}

	restored, err := svc.RestoreTurf(ctx, turf.ID, adminUserID)
	if err != nil || restored.Status != StatusApproved {
		t.Fatalf("restore: got %+v, err %v", restored, err)
	}
}

func TestModerateTurfRequestValidate(t *testing.T) {
	tests := []struct {
		name       string
		reason     string
		wantFields string
	}{
		{"valid", "A clear, specific reason.", ""},
		{"empty", "", "reason"},
		{"whitespace only", "   ", "reason"},
		{"too short", "no", "reason"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := ModerateTurfRequest{Reason: tc.reason}
			req.Normalise()

			if got := fieldNames(req.Validate()); got != tc.wantFields {
				t.Errorf("fields = %q, want %q", got, tc.wantFields)
			}
		})
	}
}
