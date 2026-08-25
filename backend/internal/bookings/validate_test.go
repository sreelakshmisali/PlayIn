package bookings

import "testing"

func TestCreateBookingRequestNormalise(t *testing.T) {
	req := CreateBookingRequest{TurfSlotID: "  slot-1  "}
	req.Normalise()

	if req.TurfSlotID != "slot-1" {
		t.Errorf("TurfSlotID = %q, want it trimmed", req.TurfSlotID)
	}
}

func TestCreateBookingRequestAccepts(t *testing.T) {
	req := CreateBookingRequest{TurfSlotID: "slot-1"}
	if problems := req.Validate(); len(problems) > 0 {
		t.Errorf("Validate() = %v, want no problems", problems)
	}
}

func TestCreateBookingRequestRejectsEmptySlotID(t *testing.T) {
	req := CreateBookingRequest{TurfSlotID: ""}
	problems := req.Validate()
	if len(problems) != 1 || problems[0].Field != "turf_slot_id" {
		t.Errorf("Validate() = %v, want one problem on turf_slot_id", problems)
	}
}
