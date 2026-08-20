package owners

import (
	"context"
	"errors"
	"testing"
)

func TestServiceProfileNotFound(t *testing.T) {
	svc, _ := newTestService()

	if _, err := svc.Profile(context.Background(), "user-nobody"); !errors.Is(err, ErrOwnerProfileNotFound) {
		t.Errorf("Profile() error = %v, want ErrOwnerProfileNotFound", err)
	}
}

func TestServiceSaveProfileCreatesThenReplaces(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := validSaveProfileRequest()
	req.Normalise()

	_, created, err := svc.SaveProfile(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if !created {
		t.Error("created = false on the first save, want true")
	}

	// PUT is a full representation: fields left out are cleared, not kept.
	second := SaveProfileRequest{DisplayName: "Turf Co"}
	second.Normalise()

	replaced, created, err := svc.SaveProfile(ctx, "user-1", second)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if created {
		t.Error("created = true on the second save, want false")
	}
	if replaced.Phone != "" || replaced.Description != "" {
		t.Errorf("profile = %+v, want the omitted fields cleared", replaced)
	}
}

func TestServicePatchProfileLeavesAbsentFieldsAlone(t *testing.T) {
	svc, _ := newTestService()
	seedProfile(t, svc, "user-1")

	var req PatchProfileRequest
	req.Phone.Set = true
	req.Phone.Value = "+91 90000 00000"

	patched, err := svc.PatchProfile(context.Background(), "user-1", req)
	if err != nil {
		t.Fatalf("PatchProfile() returned error: %v", err)
	}
	if patched.Phone != "+91 90000 00000" {
		t.Errorf("Phone = %q, want the new number", patched.Phone)
	}
	if patched.DisplayName != "Kochi Sports Arena" {
		t.Errorf("DisplayName = %q, want it untouched", patched.DisplayName)
	}
}

func TestServicePatchProfileRequiresAProfile(t *testing.T) {
	svc, _ := newTestService()

	var req PatchProfileRequest
	req.Phone.Set = true
	req.Phone.Value = "+91 90000 00000"

	if _, err := svc.PatchProfile(context.Background(), "user-1", req); !errors.Is(err, ErrOwnerProfileNotFound) {
		t.Errorf("PatchProfile() error = %v, want ErrOwnerProfileNotFound", err)
	}
}

func TestServiceSurfacesStoreFailures(t *testing.T) {
	svc, store := newTestService()
	boom := errors.New("database is on fire")
	store.failWith = boom

	if _, err := svc.Profile(context.Background(), "user-1"); !errors.Is(err, boom) {
		t.Errorf("Profile() error = %v, want the store error", err)
	}
	req := validSaveProfileRequest()
	if _, _, err := svc.SaveProfile(context.Background(), "user-1", req); !errors.Is(err, boom) {
		t.Errorf("SaveProfile() error = %v, want the store error", err)
	}
}
