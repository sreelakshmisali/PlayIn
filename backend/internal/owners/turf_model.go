package owners

import (
	"errors"
	"time"
)

// Status is a turf's place in the approval workflow. The set is closed and
// mirrors the turfs_status_chk constraint in migration 000004.
type Status string

const (
	// StatusDraft is where every turf starts: visible only to its owner,
	// editable freely, not yet asking for review.
	StatusDraft Status = "DRAFT"
	// StatusPendingApproval means the owner has submitted it for review.
	// Reaching this state is the only thing this phase's API can do; leaving
	// it is Phase 4's admin surface.
	StatusPendingApproval Status = "PENDING_APPROVAL"
	// StatusApproved means the turf is publicly listed.
	StatusApproved Status = "APPROVED"
	// StatusRejected means an admin declined it. An owner may edit and
	// resubmit.
	StatusRejected Status = "REJECTED"
	// StatusSuspended means an admin pulled a previously approved listing.
	StatusSuspended Status = "SUSPENDED"
)

// SportRef is the sports-catalogue projection a turf carries: enough to
// display and to address the sport by id, nothing more. Positions are a
// player concept and do not apply to a turf.
type SportRef struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Amenity is one entry in the amenities catalogue.
type Amenity struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// TurfImage is one photo URL attached to a turf.
type TurfImage struct {
	ID       string `json:"id"`
	ImageURL string `json:"image_url"`
}

// Turf is a listing. This is the only shape the package ever serialises for a
// turf, whether the reader is its owner or the public: the same projection
// serves both, because nothing in it needs withholding from either.
//
// OwnerDisplayName is the one field drawn from another table. It is the owner
// profile's display name, joined in for a legitimate listing to show who runs
// it; nothing from users is ever reachable through it.
type Turf struct {
	ID               string   `json:"id"`
	OwnerDisplayName string   `json:"owner_display_name"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Address          string   `json:"address"`
	City             string   `json:"city"`
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`
	Capacity         *int32   `json:"capacity,omitempty"`
	OpeningTime      string   `json:"opening_time"`
	ClosingTime      string   `json:"closing_time"`
	Status           Status   `json:"status"`
	// ModerationReason explains a REJECTED or SUSPENDED status. It is empty for
	// every other status, and reaches the owner through the same read paths
	// they already use (their turf list and turf detail) rather than a
	// separate endpoint.
	ModerationReason string      `json:"moderation_reason,omitempty"`
	Sports           []SportRef  `json:"sports"`
	Amenities        []Amenity   `json:"amenities"`
	Images           []TurfImage `json:"images"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// turfRow is the stored turf, including the surrogate owner key. It never
// leaves the package.
type turfRow struct {
	ID               string
	OwnerID          string
	OwnerDisplayName string
	Name             string
	Description      string
	Address          string
	City             string
	Latitude         *float64
	Longitude        *float64
	Capacity         *int32
	OpeningTime      string
	ClosingTime      string
	Status           Status
	ModerationReason string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (t turfRow) toTurf(sports []SportRef, amenities []Amenity, images []TurfImage) Turf {
	if sports == nil {
		sports = []SportRef{}
	}
	if amenities == nil {
		amenities = []Amenity{}
	}
	if images == nil {
		images = []TurfImage{}
	}

	return Turf{
		ID:               t.ID,
		OwnerDisplayName: t.OwnerDisplayName,
		Name:             t.Name,
		Description:      t.Description,
		Address:          t.Address,
		City:             t.City,
		Latitude:         t.Latitude,
		Longitude:        t.Longitude,
		Capacity:         t.Capacity,
		OpeningTime:      t.OpeningTime,
		ClosingTime:      t.ClosingTime,
		Status:           t.Status,
		ModerationReason: t.ModerationReason,
		Sports:           sports,
		Amenities:        amenities,
		Images:           images,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

// turfFields are the writable columns of a turf's own row: everything an
// owner edits directly, as opposed to the sports, amenities and images that
// are managed through their own sub-resources.
type turfFields struct {
	Name        string
	Description string
	Address     string
	City        string
	Latitude    *float64
	Longitude   *float64
	Capacity    *int32
	OpeningTime string
	ClosingTime string
}

// Errors returned by the service. Handlers map these to status codes; nothing
// downstream branches on error text.
var (
	// ErrTurfNotFound covers both a turf that does not exist and a turf that
	// belongs to a different owner. The two are answered identically, on
	// purpose: an owner-scoped lookup must not confirm that some other
	// owner's turf exists at all.
	ErrTurfNotFound = errors.New("owners: turf not found")
	// ErrTurfNameTaken means this owner already has a turf with this name.
	ErrTurfNameTaken = errors.New("owners: turf name already in use")
	// ErrInvalidStatusTransition means a status-changing action (submit,
	// approve, reject, suspend, restore) was attempted from a status that
	// action does not accept.
	ErrInvalidStatusTransition = errors.New("owners: turf cannot change status from its current status")
	// ErrSportNotFound means the sport id is unknown or the sport is retired.
	ErrSportNotFound = errors.New("owners: sport not found")
	// ErrTurfSportNotFound means the turf does not have this sport attached.
	ErrTurfSportNotFound = errors.New("owners: turf does not have this sport")
	// ErrAmenityNotFound means the amenity id is unknown or retired.
	ErrAmenityNotFound = errors.New("owners: amenity not found")
	// ErrTurfAmenityNotFound means the turf does not have this amenity attached.
	ErrTurfAmenityNotFound = errors.New("owners: turf does not have this amenity")
	// ErrTurfImageNotFound means the turf has no image with this id.
	ErrTurfImageNotFound = errors.New("owners: turf image not found")
	// ErrTooManyImages means the turf already carries the maximum number of
	// images this phase allows.
	ErrTooManyImages = errors.New("owners: turf already has the maximum number of images")
)
