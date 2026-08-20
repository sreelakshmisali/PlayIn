package players

import (
	"context"
	"errors"
	"testing"
)

func TestServiceSports(t *testing.T) {
	svc, _ := newTestService()

	sports, err := svc.Sports(context.Background())
	if err != nil {
		t.Fatalf("Sports() returned error: %v", err)
	}
	if len(sports) != 3 {
		t.Fatalf("Sports() returned %d sports, want 3 active ones", len(sports))
	}
	if sports[0].Name != "Badminton" {
		t.Errorf("sports[0] = %q, want Badminton first by name", sports[0].Name)
	}
	// A retired sport is not offered.
	for _, s := range sports {
		if s.ID == retiredID {
			t.Error("Sports() returned a retired sport")
		}
	}
}

// Badminton has no positions, which is what makes "position where applicable"
// a property of the data rather than a rule in the client.
func TestServiceSportsCarryPositions(t *testing.T) {
	svc, _ := newTestService()

	sports, err := svc.Sports(context.Background())
	if err != nil {
		t.Fatalf("Sports() returned error: %v", err)
	}

	byName := map[string]Sport{}
	for _, s := range sports {
		byName[s.Name] = s
	}

	if len(byName["Badminton"].Positions) != 0 {
		t.Errorf("Badminton positions = %v, want none", byName["Badminton"].Positions)
	}
	if len(byName["Football"].Positions) != 4 {
		t.Errorf("Football positions = %v, want 4", byName["Football"].Positions)
	}
}

func TestServiceProfileNotFound(t *testing.T) {
	svc, _ := newTestService()

	if _, err := svc.Profile(context.Background(), "user-nobody"); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("Profile() error = %v, want ErrProfileNotFound", err)
	}
}

func TestServiceSaveProfileCreatesThenReplaces(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := validSaveRequest()
	req.Normalise()

	first, created, err := svc.SaveProfile(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if !created {
		t.Error("created = false on the first save, want true")
	}
	if first.DisplayName != "Priya Raman" {
		t.Errorf("DisplayName = %q, want Priya Raman", first.DisplayName)
	}
	if first.Sports == nil {
		t.Error("Sports is nil, want an empty slice so it serialises as []")
	}

	// PUT is a full representation: fields left out are cleared, not kept.
	second := SaveProfileRequest{DisplayName: "Priya R"}
	second.Normalise()

	replaced, created, err := svc.SaveProfile(ctx, "user-1", second)
	if err != nil {
		t.Fatalf("SaveProfile() returned error: %v", err)
	}
	if created {
		t.Error("created = true on the second save, want false")
	}
	if replaced.Bio != "" || replaced.Location != "" || replaced.ImageURL != "" {
		t.Errorf("profile = %+v, want the omitted fields cleared", replaced)
	}
}

func TestServicePatchProfileLeavesAbsentFieldsAlone(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	seedProfile(t, svc, "user-1")

	var req PatchProfileRequest
	req.Location.Set = true
	req.Location.Value = "Chennai"

	patched, err := svc.PatchProfile(ctx, "user-1", req)
	if err != nil {
		t.Fatalf("PatchProfile() returned error: %v", err)
	}
	if patched.Location != "Chennai" {
		t.Errorf("Location = %q, want Chennai", patched.Location)
	}
	if patched.Bio != "Weekend midfielder." {
		t.Errorf("Bio = %q, want it untouched", patched.Bio)
	}
	if patched.DisplayName != "Priya Raman" {
		t.Errorf("DisplayName = %q, want it untouched", patched.DisplayName)
	}
}

// An explicit null clears the field. Without the Optional type this would be
// indistinguishable from omitting it.
func TestServicePatchProfileClearsOnNull(t *testing.T) {
	svc, _ := newTestService()
	seedProfile(t, svc, "user-1")

	var req PatchProfileRequest
	req.Bio.Set = true
	req.Bio.Null = true

	patched, err := svc.PatchProfile(context.Background(), "user-1", req)
	if err != nil {
		t.Fatalf("PatchProfile() returned error: %v", err)
	}
	if patched.Bio != "" {
		t.Errorf("Bio = %q, want it cleared", patched.Bio)
	}
	if patched.Location != "Kochi" {
		t.Errorf("Location = %q, want it untouched", patched.Location)
	}
}

func TestServicePatchProfileRequiresAProfile(t *testing.T) {
	svc, _ := newTestService()

	var req PatchProfileRequest
	req.Bio.Set = true
	req.Bio.Value = "hello"

	if _, err := svc.PatchProfile(context.Background(), "user-1", req); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("PatchProfile() error = %v, want ErrProfileNotFound", err)
	}
}

func TestServiceSetSport(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	seedProfile(t, svc, "user-1")

	profile, err := svc.SetSport(ctx, "user-1", SetSportRequest{SportID: footballID, Position: "Midfielder"})
	if err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}
	if len(profile.Sports) != 1 {
		t.Fatalf("Sports = %v, want 1 entry", profile.Sports)
	}
	if profile.Sports[0].Sport.Name != "Football" || profile.Sports[0].Position != "Midfielder" {
		t.Errorf("Sports[0] = %+v, want Football/Midfielder", profile.Sports[0])
	}
}

// Choosing a sport does not oblige a player to name a position.
func TestServiceSetSportWithoutPosition(t *testing.T) {
	svc, _ := newTestService()
	seedProfile(t, svc, "user-1")

	profile, err := svc.SetSport(context.Background(), "user-1", SetSportRequest{SportID: badmintonID})
	if err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}
	if profile.Sports[0].Position != "" {
		t.Errorf("Position = %q, want empty", profile.Sports[0].Position)
	}
}

// Repeating the call changes the position rather than adding a duplicate, which
// is what keeps the client to one verb.
func TestServiceSetSportIsAnUpsert(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	seedProfile(t, svc, "user-1")

	if _, err := svc.SetSport(ctx, "user-1", SetSportRequest{SportID: footballID, Position: "Defender"}); err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}
	profile, err := svc.SetSport(ctx, "user-1", SetSportRequest{SportID: footballID, Position: "Forward"})
	if err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}

	if len(profile.Sports) != 1 {
		t.Fatalf("Sports = %v, want 1 entry after a repeat", profile.Sports)
	}
	if profile.Sports[0].Position != "Forward" {
		t.Errorf("Position = %q, want Forward", profile.Sports[0].Position)
	}
}

func TestServiceSetSportRejections(t *testing.T) {
	tests := []struct {
		name    string
		req     SetSportRequest
		wantErr error
	}{
		{"unknown sport", SetSportRequest{SportID: "sport-nope"}, ErrSportNotFound},
		{"retired sport", SetSportRequest{SportID: retiredID, Position: "Raider"}, ErrSportNotFound},
		{"position from another sport", SetSportRequest{SportID: footballID, Position: "Wicketkeeper"}, ErrInvalidPosition},
		{"invented position", SetSportRequest{SportID: cricketID, Position: "Striker"}, ErrInvalidPosition},
		{"position on a sport that has none", SetSportRequest{SportID: badmintonID, Position: "Smasher"}, ErrInvalidPosition},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newTestService()
			seedProfile(t, svc, "user-1")

			_, err := svc.SetSport(context.Background(), "user-1", tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("SetSport() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// A rejected position has to say what is allowed, or the client is guessing.
func TestServiceSetSportPositionErrorListsChoices(t *testing.T) {
	svc, _ := newTestService()
	seedProfile(t, svc, "user-1")

	_, err := svc.SetSport(context.Background(), "user-1", SetSportRequest{SportID: footballID, Position: "Striker"})

	var positionErr *PositionError
	if !errors.As(err, &positionErr) {
		t.Fatalf("SetSport() error = %v, want a *PositionError", err)
	}

	got := positionErr.Message()
	want := "Position must be one of: Goalkeeper, Defender, Midfielder or Forward."
	if got != want {
		t.Errorf("Message() = %q, want %q", got, want)
	}
}

func TestServiceSetSportPositionErrorForPositionlessSport(t *testing.T) {
	svc, _ := newTestService()
	seedProfile(t, svc, "user-1")

	_, err := svc.SetSport(context.Background(), "user-1", SetSportRequest{SportID: badmintonID, Position: "Smasher"})

	var positionErr *PositionError
	if !errors.As(err, &positionErr) {
		t.Fatalf("SetSport() error = %v, want a *PositionError", err)
	}
	if got := positionErr.Message(); got != "Badminton does not use positions." {
		t.Errorf("Message() = %q, want the no-positions message", got)
	}
}

func TestServiceSetSportRequiresAProfile(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.SetSport(context.Background(), "user-1", SetSportRequest{SportID: footballID})
	if !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("SetSport() error = %v, want ErrProfileNotFound", err)
	}
}

func TestServiceRemoveSport(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	seedProfile(t, svc, "user-1")

	if _, err := svc.SetSport(ctx, "user-1", SetSportRequest{SportID: footballID}); err != nil {
		t.Fatalf("SetSport() returned error: %v", err)
	}

	profile, err := svc.RemoveSport(ctx, "user-1", footballID)
	if err != nil {
		t.Fatalf("RemoveSport() returned error: %v", err)
	}
	if len(profile.Sports) != 0 {
		t.Errorf("Sports = %v, want none left", profile.Sports)
	}
}

// Removing a sport the player never chose is a client mistake worth naming, not
// a quiet success.
func TestServiceRemoveSportNotPreferred(t *testing.T) {
	svc, _ := newTestService()
	seedProfile(t, svc, "user-1")

	_, err := svc.RemoveSport(context.Background(), "user-1", cricketID)
	if !errors.Is(err, ErrSportNotPreferred) {
		t.Errorf("RemoveSport() error = %v, want ErrSportNotPreferred", err)
	}
}

func TestServiceSurfacesStoreFailures(t *testing.T) {
	svc, store := newTestService()
	boom := errors.New("database is on fire")
	store.failWith = boom

	if _, err := svc.Sports(context.Background()); !errors.Is(err, boom) {
		t.Errorf("Sports() error = %v, want the store error", err)
	}
	if _, err := svc.Profile(context.Background(), "user-1"); !errors.Is(err, boom) {
		t.Errorf("Profile() error = %v, want the store error", err)
	}
}
