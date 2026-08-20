package owners

import (
	"context"
	"errors"
	"testing"
)

func TestServiceCreateTurfRequiresOwnerProfile(t *testing.T) {
	svc, _ := newTestService()

	req := validSaveTurfRequest()
	req.Normalise()

	if _, err := svc.CreateTurf(context.Background(), "user-1", req); !errors.Is(err, ErrOwnerProfileNotFound) {
		t.Errorf("CreateTurf() error = %v, want ErrOwnerProfileNotFound", err)
	}
}

func TestServiceCreateTurfStartsAsDraft(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	if turf.Status != StatusDraft {
		t.Errorf("Status = %q, want DRAFT", turf.Status)
	}
	if turf.OwnerDisplayName != "Kochi Sports Arena" {
		t.Errorf("OwnerDisplayName = %q, want the owner's display name", turf.OwnerDisplayName)
	}
	if turf.Sports == nil || turf.Amenities == nil || turf.Images == nil {
		t.Errorf("turf = %+v, want empty slices rather than nil", turf)
	}
}

func TestServiceCreateTurfRejectsDuplicateName(t *testing.T) {
	svc, store := newTestService()
	seedTurf(t, svc, store, "user-1")

	req := validSaveTurfRequest()
	req.Normalise()

	if _, err := svc.CreateTurf(context.Background(), "user-1", req); !errors.Is(err, ErrTurfNameTaken) {
		t.Errorf("CreateTurf() error = %v, want ErrTurfNameTaken", err)
	}
}

// Two different owners can each use the same turf name; the constraint is
// scoped per owner, not global.
func TestServiceCreateTurfAllowsSameNameAcrossOwners(t *testing.T) {
	svc, store := newTestService()
	seedTurf(t, svc, store, "user-1")
	seedProfile(t, svc, "user-2")

	req := validSaveTurfRequest()
	req.Normalise()

	if _, err := svc.CreateTurf(context.Background(), "user-2", req); err != nil {
		t.Errorf("CreateTurf() for a different owner returned error: %v", err)
	}
}

func TestServiceMyTurfsOnlyListsOwnTurfs(t *testing.T) {
	svc, store := newTestService()
	seedTurf(t, svc, store, "user-1")

	req := validSaveTurfRequest()
	req.Name = "A Different Turf"
	req.Normalise()
	seedProfile(t, svc, "user-2")
	if _, err := svc.CreateTurf(context.Background(), "user-2", req); err != nil {
		t.Fatalf("CreateTurf() returned error: %v", err)
	}

	mine, err := svc.MyTurfs(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("MyTurfs() returned error: %v", err)
	}
	if len(mine) != 1 {
		t.Fatalf("MyTurfs() = %d turfs, want 1", len(mine))
	}
}

// This is the core ownership guarantee: one owner must not be able to read,
// modify or delete another owner's turf. Every mutation is checked here
// against a second owner's turf id.
func TestServiceOwnershipIsolation(t *testing.T) {
	svc, store := newTestService()
	theirs := seedTurf(t, svc, store, "user-1")
	seedProfile(t, svc, "user-2")

	req := validSaveTurfRequest()
	req.Name = "Someone Else's Turf"
	req.Normalise()

	if _, err := svc.MyTurf(context.Background(), "user-2", theirs.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("MyTurf() error = %v, want ErrTurfNotFound", err)
	}
	if _, err := svc.UpdateTurf(context.Background(), "user-2", theirs.ID, req); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("UpdateTurf() error = %v, want ErrTurfNotFound", err)
	}
	if err := svc.DeleteTurf(context.Background(), "user-2", theirs.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("DeleteTurf() error = %v, want ErrTurfNotFound", err)
	}
	if _, err := svc.SubmitTurf(context.Background(), "user-2", theirs.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("SubmitTurf() error = %v, want ErrTurfNotFound", err)
	}
	if _, err := svc.SetTurfSport(context.Background(), "user-2", theirs.ID, SetTurfSportRequest{SportID: footballID}); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("SetTurfSport() error = %v, want ErrTurfNotFound", err)
	}

	// The turf must be untouched by every rejected attempt.
	unchanged, err := svc.MyTurf(context.Background(), "user-1", theirs.ID)
	if err != nil {
		t.Fatalf("MyTurf() as the real owner returned error: %v", err)
	}
	if unchanged.Name != theirs.Name {
		t.Errorf("Name = %q, want it unchanged at %q", unchanged.Name, theirs.Name)
	}
}

func TestServiceUpdateTurf(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	req := validSaveTurfRequest()
	req.Name = "Renamed Turf"
	req.Normalise()

	updated, err := svc.UpdateTurf(context.Background(), "user-1", turf.ID, req)
	if err != nil {
		t.Fatalf("UpdateTurf() returned error: %v", err)
	}
	if updated.Name != "Renamed Turf" {
		t.Errorf("Name = %q, want Renamed Turf", updated.Name)
	}
}

// Editing an APPROVED turf must drop it back to PENDING_APPROVAL: an approved
// listing that could be silently edited after review would make the approval
// meaningless.
func TestServiceUpdateTurfRevertsApprovedToPending(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")
	store.setStatus(turf.ID, StatusApproved)

	req := validSaveTurfRequest()
	req.Normalise()

	updated, err := svc.UpdateTurf(context.Background(), "user-1", turf.ID, req)
	if err != nil {
		t.Fatalf("UpdateTurf() returned error: %v", err)
	}
	if updated.Status != StatusPendingApproval {
		t.Errorf("Status = %q, want PENDING_APPROVAL after editing an approved turf", updated.Status)
	}
}

// Editing a turf in any other status must not change its status as a side
// effect; only APPROVED reverts.
func TestServiceUpdateTurfLeavesOtherStatusesAlone(t *testing.T) {
	for _, status := range []Status{StatusDraft, StatusPendingApproval, StatusRejected, StatusSuspended} {
		t.Run(string(status), func(t *testing.T) {
			svc, store := newTestService()
			turf := seedTurf(t, svc, store, "user-1")
			store.setStatus(turf.ID, status)

			req := validSaveTurfRequest()
			req.Normalise()

			updated, err := svc.UpdateTurf(context.Background(), "user-1", turf.ID, req)
			if err != nil {
				t.Fatalf("UpdateTurf() returned error: %v", err)
			}
			if updated.Status != status {
				t.Errorf("Status = %q, want it to stay %q", updated.Status, status)
			}
		})
	}
}

func TestServiceDeleteTurf(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	if err := svc.DeleteTurf(context.Background(), "user-1", turf.ID); err != nil {
		t.Fatalf("DeleteTurf() returned error: %v", err)
	}
	if _, err := svc.MyTurf(context.Background(), "user-1", turf.ID); !errors.Is(err, ErrTurfNotFound) {
		t.Errorf("MyTurf() after delete error = %v, want ErrTurfNotFound", err)
	}
}

func TestServiceSubmitTurf(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	submitted, err := svc.SubmitTurf(context.Background(), "user-1", turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}
	if submitted.Status != StatusPendingApproval {
		t.Errorf("Status = %q, want PENDING_APPROVAL", submitted.Status)
	}
}

func TestServiceSubmitTurfFromRejected(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")
	store.setStatus(turf.ID, StatusRejected)

	submitted, err := svc.SubmitTurf(context.Background(), "user-1", turf.ID)
	if err != nil {
		t.Fatalf("SubmitTurf() returned error: %v", err)
	}
	if submitted.Status != StatusPendingApproval {
		t.Errorf("Status = %q, want PENDING_APPROVAL", submitted.Status)
	}
}

func TestServiceSubmitTurfRejectsWrongStatus(t *testing.T) {
	for _, status := range []Status{StatusPendingApproval, StatusApproved, StatusSuspended} {
		t.Run(string(status), func(t *testing.T) {
			svc, store := newTestService()
			turf := seedTurf(t, svc, store, "user-1")
			store.setStatus(turf.ID, status)

			if _, err := svc.SubmitTurf(context.Background(), "user-1", turf.ID); !errors.Is(err, ErrInvalidStatusTransition) {
				t.Errorf("SubmitTurf() error = %v, want ErrInvalidStatusTransition", err)
			}
		})
	}
}

func TestServiceTurfSportLifecycle(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	added, err := svc.SetTurfSport(context.Background(), "user-1", turf.ID, SetTurfSportRequest{SportID: footballID})
	if err != nil {
		t.Fatalf("SetTurfSport() returned error: %v", err)
	}
	if len(added.Sports) != 1 || added.Sports[0].Slug != "football" {
		t.Fatalf("Sports = %+v, want one football entry", added.Sports)
	}

	// Adding the same sport again is a harmless no-op, not an error: a turf
	// either supports a sport or it does not, there is nothing to upsert.
	again, err := svc.SetTurfSport(context.Background(), "user-1", turf.ID, SetTurfSportRequest{SportID: footballID})
	if err != nil {
		t.Fatalf("SetTurfSport() repeat returned error: %v", err)
	}
	if len(again.Sports) != 1 {
		t.Errorf("Sports = %+v, want still one entry after a repeat", again.Sports)
	}

	removed, err := svc.RemoveTurfSport(context.Background(), "user-1", turf.ID, footballID)
	if err != nil {
		t.Fatalf("RemoveTurfSport() returned error: %v", err)
	}
	if len(removed.Sports) != 0 {
		t.Errorf("Sports = %+v, want none left", removed.Sports)
	}
}

func TestServiceSetTurfSportRejectsUnknownSport(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	_, err := svc.SetTurfSport(context.Background(), "user-1", turf.ID, SetTurfSportRequest{SportID: "sport-nope"})
	if !errors.Is(err, ErrSportNotFound) {
		t.Errorf("SetTurfSport() error = %v, want ErrSportNotFound", err)
	}
}

func TestServiceRemoveTurfSportNotAttached(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	_, err := svc.RemoveTurfSport(context.Background(), "user-1", turf.ID, cricketID)
	if !errors.Is(err, ErrTurfSportNotFound) {
		t.Errorf("RemoveTurfSport() error = %v, want ErrTurfSportNotFound", err)
	}
}

func TestServiceTurfAmenityLifecycle(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	added, err := svc.SetTurfAmenity(context.Background(), "user-1", turf.ID, SetTurfAmenityRequest{AmenityID: parkingID})
	if err != nil {
		t.Fatalf("SetTurfAmenity() returned error: %v", err)
	}
	if len(added.Amenities) != 1 || added.Amenities[0].Slug != "parking" {
		t.Fatalf("Amenities = %+v, want one parking entry", added.Amenities)
	}

	removed, err := svc.RemoveTurfAmenity(context.Background(), "user-1", turf.ID, parkingID)
	if err != nil {
		t.Fatalf("RemoveTurfAmenity() returned error: %v", err)
	}
	if len(removed.Amenities) != 0 {
		t.Errorf("Amenities = %+v, want none left", removed.Amenities)
	}
}

func TestServiceSetTurfAmenityRejectsUnknownAmenity(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	_, err := svc.SetTurfAmenity(context.Background(), "user-1", turf.ID, SetTurfAmenityRequest{AmenityID: "amenity-nope"})
	if !errors.Is(err, ErrAmenityNotFound) {
		t.Errorf("SetTurfAmenity() error = %v, want ErrAmenityNotFound", err)
	}
}

func TestServiceTurfImageLifecycle(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	added, err := svc.AddTurfImage(context.Background(), "user-1", turf.ID, AddTurfImageRequest{ImageURL: "https://cdn.playhub.test/1.jpg"})
	if err != nil {
		t.Fatalf("AddTurfImage() returned error: %v", err)
	}
	if len(added.Images) != 1 {
		t.Fatalf("Images = %+v, want one entry", added.Images)
	}

	removed, err := svc.RemoveTurfImage(context.Background(), "user-1", turf.ID, added.Images[0].ID)
	if err != nil {
		t.Fatalf("RemoveTurfImage() returned error: %v", err)
	}
	if len(removed.Images) != 0 {
		t.Errorf("Images = %+v, want none left", removed.Images)
	}
}

func TestServiceAddTurfImageEnforcesCap(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	for i := 0; i < maxTurfImages; i++ {
		req := AddTurfImageRequest{ImageURL: "https://cdn.playhub.test/" + string(rune('a'+i)) + ".jpg"}
		if _, err := svc.AddTurfImage(context.Background(), "user-1", turf.ID, req); err != nil {
			t.Fatalf("AddTurfImage() #%d returned error: %v", i, err)
		}
	}

	_, err := svc.AddTurfImage(context.Background(), "user-1", turf.ID, AddTurfImageRequest{ImageURL: "https://cdn.playhub.test/over.jpg"})
	if !errors.Is(err, ErrTooManyImages) {
		t.Errorf("AddTurfImage() over the cap error = %v, want ErrTooManyImages", err)
	}
}

func TestServiceRemoveTurfImageNotFound(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	_, err := svc.RemoveTurfImage(context.Background(), "user-1", turf.ID, "image-nope")
	if !errors.Is(err, ErrTurfImageNotFound) {
		t.Errorf("RemoveTurfImage() error = %v, want ErrTurfImageNotFound", err)
	}
}

func TestServicePublicTurfsOnlyShowsApproved(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	// DRAFT: not listed.
	if turfs, err := svc.PublicTurfs(context.Background()); err != nil || len(turfs) != 0 {
		t.Fatalf("PublicTurfs() with a DRAFT turf = %v, %v, want none", turfs, err)
	}

	for _, status := range []Status{StatusPendingApproval, StatusRejected, StatusSuspended} {
		store.setStatus(turf.ID, status)
		if turfs, err := svc.PublicTurfs(context.Background()); err != nil || len(turfs) != 0 {
			t.Errorf("PublicTurfs() with status %s = %v, %v, want none", status, turfs, err)
		}
		if _, err := svc.PublicTurf(context.Background(), turf.ID); !errors.Is(err, ErrTurfNotFound) {
			t.Errorf("PublicTurf() with status %s error = %v, want ErrTurfNotFound", status, err)
		}
	}

	store.setStatus(turf.ID, StatusApproved)
	turfs, err := svc.PublicTurfs(context.Background())
	if err != nil {
		t.Fatalf("PublicTurfs() returned error: %v", err)
	}
	if len(turfs) != 1 {
		t.Fatalf("PublicTurfs() = %d turfs, want 1 once approved", len(turfs))
	}

	got, err := svc.PublicTurf(context.Background(), turf.ID)
	if err != nil {
		t.Fatalf("PublicTurf() returned error: %v", err)
	}
	if got.ID != turf.ID {
		t.Errorf("ID = %q, want %q", got.ID, turf.ID)
	}
}

// A guessed id for a turf that exists but is not approved must not confirm
// its existence: the response is identical to an id that does not exist.
func TestServicePublicTurfDoesNotLeakUnapprovedExistence(t *testing.T) {
	svc, store := newTestService()
	turf := seedTurf(t, svc, store, "user-1")

	_, realErr := svc.PublicTurf(context.Background(), turf.ID)
	_, fakeErr := svc.PublicTurf(context.Background(), "turf-does-not-exist")

	if !errors.Is(realErr, ErrTurfNotFound) || !errors.Is(fakeErr, ErrTurfNotFound) {
		t.Errorf("errors = %v, %v, want both ErrTurfNotFound", realErr, fakeErr)
	}
}
