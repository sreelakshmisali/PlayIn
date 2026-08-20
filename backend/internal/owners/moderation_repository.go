package owners

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PendingTurfs lists turfs awaiting review, oldest submission first: that is
// the order a moderation queue should be worked in.
func (r *Repository) PendingTurfs(ctx context.Context) ([]Turf, error) {
	const query = `
		SELECT ` + turfColumns + ` ` + turfFrom + `
		WHERE t.status = 'PENDING_APPROVAL'
		ORDER BY t.created_at`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("select pending turfs: %w", err)
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
		return nil, fmt.Errorf("iterate pending turfs: %w", err)
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

// TurfByID reads any turf regardless of owner or status. Unlike
// TurfByOwnerAndID, this is not scoped to a caller: it exists for the admin
// surface, where seeing any turf by id is the entire point.
func (r *Repository) TurfByID(ctx context.Context, turfID string) (Turf, error) {
	const query = `SELECT ` + turfColumns + ` ` + turfFrom + ` WHERE t.id = $1`

	row, err := scanTurfRow(r.db.QueryRow(ctx, query, turfID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Turf{}, ErrTurfNotFound
		}
		return Turf{}, fmt.Errorf("select turf: %w", err)
	}
	return r.assembleTurf(ctx, row)
}

// ApproveTurf moves a turf from PENDING_APPROVAL to APPROVED, clearing any
// stale reason from a previous rejection or suspension.
func (r *Repository) ApproveTurf(ctx context.Context, turfID, adminUserID string) (Turf, error) {
	return r.transitionTurf(ctx, turfID,
		`UPDATE turfs SET status = 'APPROVED', moderation_reason = NULL, moderated_by = $2, moderated_at = now()
		 WHERE id = $1 AND status = 'PENDING_APPROVAL'
		 RETURNING id::text`,
		adminUserID)
}

// RejectTurf moves a turf from PENDING_APPROVAL to REJECTED with a reason the
// owner can act on before resubmitting.
func (r *Repository) RejectTurf(ctx context.Context, turfID, adminUserID, reason string) (Turf, error) {
	return r.transitionTurf(ctx, turfID,
		`UPDATE turfs SET status = 'REJECTED', moderation_reason = $3, moderated_by = $2, moderated_at = now()
		 WHERE id = $1 AND status = 'PENDING_APPROVAL'
		 RETURNING id::text`,
		adminUserID, reason)
}

// SuspendTurf pulls a live listing, moving it from APPROVED to SUSPENDED with
// a reason. A suspended turf drops out of public visibility immediately: the
// status change is the only thing the public turf queries check.
func (r *Repository) SuspendTurf(ctx context.Context, turfID, adminUserID, reason string) (Turf, error) {
	return r.transitionTurf(ctx, turfID,
		`UPDATE turfs SET status = 'SUSPENDED', moderation_reason = $3, moderated_by = $2, moderated_at = now()
		 WHERE id = $1 AND status = 'APPROVED'
		 RETURNING id::text`,
		adminUserID, reason)
}

// RestoreTurf reinstates a suspended turf directly to APPROVED. The listing
// itself did not change while suspended, so restoring it does not require the
// PENDING_APPROVAL re-review a content edit does.
func (r *Repository) RestoreTurf(ctx context.Context, turfID, adminUserID string) (Turf, error) {
	return r.transitionTurf(ctx, turfID,
		`UPDATE turfs SET status = 'APPROVED', moderation_reason = NULL, moderated_by = $2, moderated_at = now()
		 WHERE id = $1 AND status = 'SUSPENDED'
		 RETURNING id::text`,
		adminUserID)
}

// transitionTurf runs one guarded status-change UPDATE and reloads the turf.
//
// The target status and the required current status are both baked into the
// caller's query text, so the write either happens atomically or not at all;
// there is no read-then-write window for the status to change underneath it.
// Zero rows affected could mean the turf does not exist or that it is not in
// the status this transition requires, and the two are told apart with a
// follow-up unscoped read, exactly as SubmitTurf does for the owner-facing
// transition.
func (r *Repository) transitionTurf(ctx context.Context, turfID, query string, args ...any) (Turf, error) {
	queryArgs := append([]any{turfID}, args...)

	var id string
	err := r.db.QueryRow(ctx, query, queryArgs...).Scan(&id)
	switch {
	case err == nil:
		return r.TurfByID(ctx, id)
	case errors.Is(err, pgx.ErrNoRows), isInvalidUUID(err):
		if _, existsErr := r.TurfByID(ctx, turfID); existsErr != nil {
			return Turf{}, ErrTurfNotFound
		}
		return Turf{}, ErrInvalidStatusTransition
	default:
		return Turf{}, fmt.Errorf("transition turf: %w", err)
	}
}
