package owners

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// publicTurfListLimit caps the unfiltered public listing. This phase has no
// pagination, so a fixed ceiling stands in for it rather than leaving the
// query unbounded.
const publicTurfListLimit = 50

// turfColumns is the projection every turf read shares. owner_display_name
// comes from the one join this package makes into another table's data, and
// it carries nothing beyond the name an owner chose to show players.
const turfColumns = `
	t.id::text, t.owner_id::text, op.display_name,
	t.name, coalesce(t.description, ''), t.address, t.city,
	t.latitude, t.longitude, t.capacity,
	t.opening_time, t.closing_time, t.status, coalesce(t.moderation_reason, ''),
	t.created_at, t.updated_at`

const turfFrom = `FROM turfs t JOIN owner_profiles op ON op.id = t.owner_id`

// CreateTurf inserts a turf under the given owner profile. It starts as DRAFT,
// the column default, so it is visible only to its owner until submitted.
func (r *Repository) CreateTurf(ctx context.Context, ownerProfileID string, f turfFields) (Turf, error) {
	const insert = `
		INSERT INTO turfs (owner_id, name, description, address, city, latitude, longitude, capacity, opening_time, closing_time)
		VALUES ($1, $2, nullif($3, ''), $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text`

	var turfID string
	err := r.db.QueryRow(ctx, insert,
		ownerProfileID, f.Name, f.Description, f.Address, f.City,
		f.Latitude, f.Longitude, f.Capacity, f.OpeningTime, f.ClosingTime,
	).Scan(&turfID)
	if err != nil {
		if isUniqueViolation(err) {
			return Turf{}, ErrTurfNameTaken
		}
		return Turf{}, fmt.Errorf("insert turf: %w", err)
	}

	return r.TurfByOwnerAndID(ctx, ownerProfileID, turfID)
}

// TurfsByOwner lists every turf an owner has, any status, newest first.
func (r *Repository) TurfsByOwner(ctx context.Context, ownerProfileID string) ([]Turf, error) {
	const query = `SELECT ` + turfColumns + ` ` + turfFrom + ` WHERE t.owner_id = $1 ORDER BY t.created_at DESC`

	rows, err := r.db.Query(ctx, query, ownerProfileID)
	if err != nil {
		return nil, fmt.Errorf("select owner turfs: %w", err)
	}
	defer rows.Close()

	turfRows := make([]turfRow, 0, 8)
	for rows.Next() {
		row, err := scanTurfRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan turf: %w", err)
		}
		turfRows = append(turfRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owner turfs: %w", err)
	}

	turfs := make([]Turf, 0, len(turfRows))
	for _, row := range turfRows {
		turf, err := r.assembleTurf(ctx, row)
		if err != nil {
			return nil, err
		}
		turfs = append(turfs, turf)
	}
	return turfs, nil
}

// TurfByOwnerAndID reads one of an owner's own turfs.
//
// A turf that does not exist and a turf that belongs to a different owner
// report the same ErrTurfNotFound. An owner-scoped lookup must not confirm
// that some other owner's turf exists at all.
func (r *Repository) TurfByOwnerAndID(ctx context.Context, ownerProfileID, turfID string) (Turf, error) {
	const query = `SELECT ` + turfColumns + ` ` + turfFrom + ` WHERE t.id = $1 AND t.owner_id = $2`

	row, err := scanTurfRow(r.db.QueryRow(ctx, query, turfID, ownerProfileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Turf{}, ErrTurfNotFound
		}
		return Turf{}, fmt.Errorf("select turf: %w", err)
	}
	return r.assembleTurf(ctx, row)
}

// UpdateTurf replaces a turf's own writable columns.
//
// Editing an APPROVED turf drops it back to PENDING_APPROVAL in the same
// statement: an approved listing that could be silently edited after review
// would make the approval meaningless. Every other status is left as it is,
// since editing a DRAFT or a REJECTED turf while preparing to submit it is
// not itself a status change.
func (r *Repository) UpdateTurf(ctx context.Context, ownerProfileID, turfID string, f turfFields) (Turf, error) {
	const query = `
		UPDATE turfs SET
			name = $3, description = nullif($4, ''), address = $5, city = $6,
			latitude = $7, longitude = $8, capacity = $9,
			opening_time = $10, closing_time = $11,
			status = CASE WHEN status = 'APPROVED' THEN 'PENDING_APPROVAL' ELSE status END
		WHERE id = $1 AND owner_id = $2
		RETURNING id::text`

	var id string
	err := r.db.QueryRow(ctx, query,
		turfID, ownerProfileID, f.Name, f.Description, f.Address, f.City,
		f.Latitude, f.Longitude, f.Capacity, f.OpeningTime, f.ClosingTime,
	).Scan(&id)
	if err != nil {
		switch {
		case isUniqueViolation(err):
			return Turf{}, ErrTurfNameTaken
		case errors.Is(err, pgx.ErrNoRows), isInvalidUUID(err):
			return Turf{}, ErrTurfNotFound
		default:
			return Turf{}, fmt.Errorf("update turf: %w", err)
		}
	}

	return r.TurfByOwnerAndID(ctx, ownerProfileID, id)
}

// DeleteTurf removes a turf. Its sports, amenities and images cascade with it.
func (r *Repository) DeleteTurf(ctx context.Context, ownerProfileID, turfID string) error {
	const query = `DELETE FROM turfs WHERE id = $1 AND owner_id = $2`

	tag, err := r.db.Exec(ctx, query, turfID, ownerProfileID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrTurfNotFound
		}
		return fmt.Errorf("delete turf: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTurfNotFound
	}
	return nil
}

// SubmitTurf moves a turf from DRAFT or REJECTED to PENDING_APPROVAL.
//
// The status condition is part of the guarded UPDATE itself, so the write
// either happens atomically or not at all; there is no read-then-write window
// for the status to change underneath it. Zero rows affected could mean the
// turf is not this owner's or that it is not in a submittable status, and the
// two are told apart with a follow-up read scoped by the same owner check
// every other lookup uses.
func (r *Repository) SubmitTurf(ctx context.Context, ownerProfileID, turfID string) (Turf, error) {
	const query = `
		UPDATE turfs SET status = 'PENDING_APPROVAL', moderation_reason = NULL,
			moderated_by = NULL, moderated_at = NULL
		WHERE id = $1 AND owner_id = $2 AND status IN ('DRAFT', 'REJECTED')
		RETURNING id::text`

	var id string
	err := r.db.QueryRow(ctx, query, turfID, ownerProfileID).Scan(&id)
	switch {
	case err == nil:
		return r.TurfByOwnerAndID(ctx, ownerProfileID, id)
	case errors.Is(err, pgx.ErrNoRows), isInvalidUUID(err):
		if _, existsErr := r.TurfByOwnerAndID(ctx, ownerProfileID, turfID); existsErr != nil {
			return Turf{}, ErrTurfNotFound
		}
		return Turf{}, ErrInvalidStatusTransition
	default:
		return Turf{}, fmt.Errorf("submit turf: %w", err)
	}
}

// SetTurfSport attaches a sport to a turf, or is a no-op if it is already
// attached.
//
// The insert selects from turfs and sports together, so both the ownership
// check and the sport's validity are enforced against the live rows inside the
// same statement. ON CONFLICT DO UPDATE, rather than DO NOTHING, means a
// repeat call still reports a row affected, which is what lets the zero-rows
// case below mean unambiguously "the source rows did not match", not "it was
// already there".
func (r *Repository) SetTurfSport(ctx context.Context, ownerProfileID, turfID, sportID string) error {
	const query = `
		INSERT INTO turf_sports (turf_id, sport_id)
		SELECT t.id, s.id
		FROM turfs t
		JOIN sports s ON s.id = $3 AND s.is_active
		WHERE t.id = $2 AND t.owner_id = $1
		ON CONFLICT (turf_id, sport_id) DO UPDATE SET sport_id = EXCLUDED.sport_id`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, sportID)
	if err != nil && !isInvalidUUID(err) {
		return fmt.Errorf("upsert turf sport: %w", err)
	}
	if err == nil && tag.RowsAffected() > 0 {
		return nil
	}

	// Disambiguate with the same owner-scoped check every other lookup uses,
	// so this cannot be used to confirm another owner's turf exists.
	if _, existsErr := r.TurfByOwnerAndID(ctx, ownerProfileID, turfID); existsErr != nil {
		return ErrTurfNotFound
	}
	return ErrSportNotFound
}

// RemoveTurfSport detaches a sport from a turf. Every failure mode, the sport
// was never attached, the turf does not exist, or it is not this owner's,
// collapses to the same ErrTurfSportNotFound: none of them can be fixed by
// retrying, and none should tell an unauthorised caller which case it was.
func (r *Repository) RemoveTurfSport(ctx context.Context, ownerProfileID, turfID, sportID string) error {
	const query = `
		DELETE FROM turf_sports USING turfs t
		WHERE turf_sports.turf_id = t.id AND t.id = $2 AND t.owner_id = $1
		  AND turf_sports.sport_id = $3`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, sportID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrTurfSportNotFound
		}
		return fmt.Errorf("delete turf sport: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTurfSportNotFound
	}
	return nil
}

// SetTurfAmenity attaches an amenity to a turf. See SetTurfSport: the same
// guarded-upsert shape, for the same reasons.
func (r *Repository) SetTurfAmenity(ctx context.Context, ownerProfileID, turfID, amenityID string) error {
	const query = `
		INSERT INTO turf_amenities (turf_id, amenity_id)
		SELECT t.id, a.id
		FROM turfs t
		JOIN amenities a ON a.id = $3 AND a.is_active
		WHERE t.id = $2 AND t.owner_id = $1
		ON CONFLICT (turf_id, amenity_id) DO UPDATE SET amenity_id = EXCLUDED.amenity_id`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, amenityID)
	if err != nil && !isInvalidUUID(err) {
		return fmt.Errorf("upsert turf amenity: %w", err)
	}
	if err == nil && tag.RowsAffected() > 0 {
		return nil
	}

	if _, existsErr := r.TurfByOwnerAndID(ctx, ownerProfileID, turfID); existsErr != nil {
		return ErrTurfNotFound
	}
	return ErrAmenityNotFound
}

// RemoveTurfAmenity detaches an amenity from a turf.
func (r *Repository) RemoveTurfAmenity(ctx context.Context, ownerProfileID, turfID, amenityID string) error {
	const query = `
		DELETE FROM turf_amenities USING turfs t
		WHERE turf_amenities.turf_id = t.id AND t.id = $2 AND t.owner_id = $1
		  AND turf_amenities.amenity_id = $3`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, amenityID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrTurfAmenityNotFound
		}
		return fmt.Errorf("delete turf amenity: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTurfAmenityNotFound
	}
	return nil
}

// AddTurfImage appends an image URL, or replaces its row if the same URL was
// already attached. The count check ahead of the insert is a soft cap, not a
// concurrency-proof one: two simultaneous adds at the limit could both pass it
// and leave the turf one image over. That is an acceptable gap for a limit
// whose purpose is discouraging an unbounded gallery, not enforcing an exact
// ceiling.
func (r *Repository) AddTurfImage(ctx context.Context, ownerProfileID, turfID, imageURL string) (TurfImage, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT count(*) FROM turf_images ti
		JOIN turfs t ON t.id = ti.turf_id
		WHERE t.id = $1 AND t.owner_id = $2`, turfID, ownerProfileID,
	).Scan(&count)
	if err != nil {
		if isInvalidUUID(err) {
			return TurfImage{}, ErrTurfNotFound
		}
		return TurfImage{}, fmt.Errorf("count turf images: %w", err)
	}
	if count >= maxTurfImages {
		return TurfImage{}, ErrTooManyImages
	}

	const insert = `
		INSERT INTO turf_images (turf_id, image_url)
		SELECT t.id, $3
		FROM turfs t
		WHERE t.id = $1 AND t.owner_id = $2
		ON CONFLICT (turf_id, image_url) DO UPDATE SET image_url = EXCLUDED.image_url
		RETURNING id::text`

	var id string
	if err := r.db.QueryRow(ctx, insert, turfID, ownerProfileID, imageURL).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return TurfImage{}, ErrTurfNotFound
		}
		return TurfImage{}, fmt.Errorf("insert turf image: %w", err)
	}
	return TurfImage{ID: id, ImageURL: imageURL}, nil
}

// RemoveTurfImage drops one image from a turf.
func (r *Repository) RemoveTurfImage(ctx context.Context, ownerProfileID, turfID, imageID string) error {
	const query = `
		DELETE FROM turf_images USING turfs t
		WHERE turf_images.turf_id = t.id AND t.id = $2 AND t.owner_id = $1
		  AND turf_images.id = $3`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, imageID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrTurfImageNotFound
		}
		return fmt.Errorf("delete turf image: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTurfImageNotFound
	}
	return nil
}

// Amenities returns the active catalogue, ordered by name.
func (r *Repository) Amenities(ctx context.Context) ([]Amenity, error) {
	const query = `SELECT id::text, slug, name FROM amenities WHERE is_active ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("select amenities: %w", err)
	}
	defer rows.Close()

	amenities := make([]Amenity, 0, 8)
	for rows.Next() {
		var a Amenity
		if err := rows.Scan(&a.ID, &a.Slug, &a.Name); err != nil {
			return nil, fmt.Errorf("scan amenity: %w", err)
		}
		amenities = append(amenities, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate amenities: %w", err)
	}
	return amenities, nil
}

// PublicTurfs lists APPROVED turfs, newest first. Every other status is
// invisible here by construction: the WHERE clause is the only thing standing
// between a DRAFT turf and the public, so it names the one status that may
// appear rather than excluding the ones that may not.
func (r *Repository) PublicTurfs(ctx context.Context) ([]Turf, error) {
	const query = `
		SELECT ` + turfColumns + ` ` + turfFrom + `
		WHERE t.status = 'APPROVED'
		ORDER BY t.created_at DESC
		LIMIT $1`

	rows, err := r.db.Query(ctx, query, publicTurfListLimit)
	if err != nil {
		return nil, fmt.Errorf("select public turfs: %w", err)
	}
	defer rows.Close()

	turfRows := make([]turfRow, 0, publicTurfListLimit)
	for rows.Next() {
		row, err := scanTurfRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan turf: %w", err)
		}
		turfRows = append(turfRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public turfs: %w", err)
	}

	turfs := make([]Turf, 0, len(turfRows))
	for _, row := range turfRows {
		turf, err := r.assembleTurf(ctx, row)
		if err != nil {
			return nil, err
		}
		turfs = append(turfs, turf)
	}
	return turfs, nil
}

// PublicTurfByID reads one turf, but only if it is APPROVED. A turf in any
// other status reports ErrTurfNotFound here, the same as one that does not
// exist at all, so a guessed id cannot be used to find a listing before it is
// approved.
func (r *Repository) PublicTurfByID(ctx context.Context, turfID string) (Turf, error) {
	const query = `SELECT ` + turfColumns + ` ` + turfFrom + ` WHERE t.id = $1 AND t.status = 'APPROVED'`

	row, err := scanTurfRow(r.db.QueryRow(ctx, query, turfID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Turf{}, ErrTurfNotFound
		}
		return Turf{}, fmt.Errorf("select public turf: %w", err)
	}
	return r.assembleTurf(ctx, row)
}

// assembleTurf loads a turf's sports, amenities and images and projects the
// row into the response shape.
func (r *Repository) assembleTurf(ctx context.Context, row turfRow) (Turf, error) {
	sports, err := r.turfSports(ctx, row.ID)
	if err != nil {
		return Turf{}, err
	}
	amenities, err := r.turfAmenities(ctx, row.ID)
	if err != nil {
		return Turf{}, err
	}
	images, err := r.turfImages(ctx, row.ID)
	if err != nil {
		return Turf{}, err
	}
	return row.toTurf(sports, amenities, images), nil
}

func (r *Repository) turfSports(ctx context.Context, turfID string) ([]SportRef, error) {
	const query = `
		SELECT s.id::text, s.slug, s.name
		FROM turf_sports ts
		JOIN sports s ON s.id = ts.sport_id
		WHERE ts.turf_id = $1
		ORDER BY s.name`

	rows, err := r.db.Query(ctx, query, turfID)
	if err != nil {
		return nil, fmt.Errorf("select turf sports: %w", err)
	}
	defer rows.Close()

	out := make([]SportRef, 0, 4)
	for rows.Next() {
		var s SportRef
		if err := rows.Scan(&s.ID, &s.Slug, &s.Name); err != nil {
			return nil, fmt.Errorf("scan turf sport: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate turf sports: %w", err)
	}
	return out, nil
}

func (r *Repository) turfAmenities(ctx context.Context, turfID string) ([]Amenity, error) {
	const query = `
		SELECT a.id::text, a.slug, a.name
		FROM turf_amenities ta
		JOIN amenities a ON a.id = ta.amenity_id
		WHERE ta.turf_id = $1
		ORDER BY a.name`

	rows, err := r.db.Query(ctx, query, turfID)
	if err != nil {
		return nil, fmt.Errorf("select turf amenities: %w", err)
	}
	defer rows.Close()

	out := make([]Amenity, 0, 4)
	for rows.Next() {
		var a Amenity
		if err := rows.Scan(&a.ID, &a.Slug, &a.Name); err != nil {
			return nil, fmt.Errorf("scan turf amenity: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate turf amenities: %w", err)
	}
	return out, nil
}

func (r *Repository) turfImages(ctx context.Context, turfID string) ([]TurfImage, error) {
	const query = `SELECT id::text, image_url FROM turf_images WHERE turf_id = $1 ORDER BY created_at, id`

	rows, err := r.db.Query(ctx, query, turfID)
	if err != nil {
		return nil, fmt.Errorf("select turf images: %w", err)
	}
	defer rows.Close()

	out := make([]TurfImage, 0, 4)
	for rows.Next() {
		var img TurfImage
		if err := rows.Scan(&img.ID, &img.ImageURL); err != nil {
			return nil, fmt.Errorf("scan turf image: %w", err)
		}
		out = append(out, img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate turf images: %w", err)
	}
	return out, nil
}

func scanTurfRow(row pgx.Row) (turfRow, error) {
	var t turfRow
	err := row.Scan(
		&t.ID, &t.OwnerID, &t.OwnerDisplayName,
		&t.Name, &t.Description, &t.Address, &t.City,
		&t.Latitude, &t.Longitude, &t.Capacity,
		&t.OpeningTime, &t.ClosingTime, &t.Status, &t.ModerationReason,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}
