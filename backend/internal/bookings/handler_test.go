package bookings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/orgmelethil/playhub/backend/internal/auth"
	"github.com/orgmelethil/playhub/backend/internal/httpx"
)

const testPrefix = "/api/v1"

// newTestMux mounts the handler exactly as the router does, so these tests
// exercise the real route table including the guards.
func newTestMux(role auth.Role) (*http.ServeMux, *memStore) {
	handler, store := newTestHandler(testUser(role))

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpx.NotFound)
	handler.Routes(mux, testPrefix)

	return mux, store
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		req.Header.Set("Authorization", "Bearer any-token")
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeBooking(t *testing.T, rec *httptest.ResponseRecorder) Booking {
	t.Helper()
	var booking Booking
	if err := json.Unmarshal(rec.Body.Bytes(), &booking); err != nil {
		t.Fatalf("decoding the booking failed: %v (body: %s)", err, rec.Body)
	}
	return booking
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()
	var envelope httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decoding the error body failed: %v (body: %s)", err, rec.Body)
	}
	return envelope.Error
}

func TestHandlerCreateRequiresAuth(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"slot-1"}`, false)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// Booking is PLAYER-only: an OWNER token is refused, not quietly allowed.
func TestHandlerCreateRejectsNonPlayerRole(t *testing.T) {
	mux, _ := newTestMux(auth.RoleOwner)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"slot-1"}`, true)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandlerCreateSucceeds(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)
	slotID := store.seedSlot(fakeSlotOpen)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"`+slotID+`"}`, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body)
	}

	booking := decodeBooking(t, rec)
	if booking.TurfSlotID != slotID {
		t.Errorf("TurfSlotID = %q, want %q", booking.TurfSlotID, slotID)
	}
	if booking.Status != StatusConfirmed {
		t.Errorf("Status = %q, want %q", booking.Status, StatusConfirmed)
	}
}

func TestHandlerCreateRejectsMissingSlotID(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":""}`, true)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestHandlerCreateRefusesNonOpenSlot(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)
	slotID := store.seedSlot(fakeSlotBooked)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"`+slotID+`"}`, true)
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if got := decodeError(t, rec).Code; got != "slot_not_bookable" {
		t.Errorf("error code = %q, want %q", got, "slot_not_bookable")
	}
}

func TestHandlerMyBookingsListsOwnBookingsOnly(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)
	slotID := store.seedSlot(fakeSlotOpen)

	do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"`+slotID+`"}`, true)

	rec := do(t, mux, http.MethodGet, testPrefix+"/players/me/bookings", "", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	var body struct {
		Bookings []Booking `json:"bookings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding the bookings list failed: %v", err)
	}
	if len(body.Bookings) != 1 {
		t.Fatalf("len(bookings) = %d, want 1", len(body.Bookings))
	}
}

func TestHandlerBookingNotFoundForAnotherPlayer(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)
	slotID := store.seedSlot(fakeSlotOpen)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"`+slotID+`"}`, true)
	booking := decodeBooking(t, rec)

	// A second mux, authenticated as a different player.
	otherHandler, _ := newTestHandler(auth.User{ID: otherPlayerID, Role: auth.RolePlayer, IsActive: true})
	otherMux := http.NewServeMux()
	otherHandler.Routes(otherMux, testPrefix)

	getRec := do(t, otherMux, http.MethodGet, testPrefix+"/players/me/bookings/"+booking.ID, "", true)
	if getRec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", getRec.Code, http.StatusNotFound)
	}
}

func TestHandlerCancelSucceeds(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)
	slotID := store.seedSlot(fakeSlotOpen)

	createRec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"`+slotID+`"}`, true)
	booking := decodeBooking(t, createRec)

	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings/"+booking.ID+"/cancel", "", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	cancelled := decodeBooking(t, rec)
	if cancelled.Status != StatusCancelled {
		t.Errorf("Status = %q, want %q", cancelled.Status, StatusCancelled)
	}
}

func TestHandlerCancelRefusesSecondCancel(t *testing.T) {
	mux, store := newTestMux(auth.RolePlayer)
	slotID := store.seedSlot(fakeSlotOpen)

	createRec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings", `{"turf_slot_id":"`+slotID+`"}`, true)
	booking := decodeBooking(t, createRec)

	do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings/"+booking.ID+"/cancel", "", true)
	rec := do(t, mux, http.MethodPost, testPrefix+"/players/me/bookings/"+booking.ID+"/cancel", "", true)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if got := decodeError(t, rec).Code; got != "booking_already_cancelled" {
		t.Errorf("error code = %q, want %q", got, "booking_already_cancelled")
	}
}

func TestHandlerMethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(auth.RolePlayer)

	rec := do(t, mux, http.MethodDelete, testPrefix+"/players/me/bookings", "", true)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
