package owners

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// slotColumns is the projection every slot read shares. available is computed
// here, not stored: OPEN and outside both kinds of block, evaluated fresh on
// every read against the current block rows.
const slotColumns = `
	s.id::text, s.slot_date, s.start_time, s.end_time, s.price, s.status,
	(s.status = 'OPEN'
		AND NOT EXISTS (
			SELECT 1 FROM turf_blocked_dates bd
			WHERE bd.turf_id = s.turf_id AND bd.blocked_date = s.slot_date
		)
		AND NOT EXISTS (
			SELECT 1 FROM turf_blocked_time_ranges btr
			WHERE btr.turf_id = s.turf_id AND btr.blocked_date = s.slot_date
			  AND btr.minute_range && s.minute_range
		)
	) AS available,
	s.created_at, s.updated_at`

// UpdateSlotSettings sets the slot duration and price a turf generates slots
// with. Existing slots keep the price they were generated with; only future
// generation is affected.
func (r *Repository) UpdateSlotSettings(ctx context.Context, ownerProfileID, turfID string, f slotSettingsFields) (Turf, error) {
	const query = `
		UPDATE turfs SET slot_duration_minutes = $3, slot_price = $4
		WHERE id = $1 AND owner_id = $2
		RETURNING id::text`

	var id string
	err := r.db.QueryRow(ctx, query, turfID, ownerProfileID, f.DurationMinutes, f.Price).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Turf{}, ErrTurfNotFound
		}
		return Turf{}, fmt.Errorf("update slot settings: %w", err)
	}
	return r.TurfByOwnerAndID(ctx, ownerProfileID, id)
}

// SlotsInRange lists an owner's own slots for one of their turfs over a date
// range, inclusive, ordered by date then start time.
func (r *Repository) SlotsInRange(ctx context.Context, ownerProfileID, turfID string, from, to time.Time) ([]Slot, error) {
	const query = `
		SELECT ` + slotColumns + `
		FROM turf_slots s
		JOIN turfs t ON t.id = s.turf_id
		WHERE s.turf_id = $1 AND t.owner_id = $2 AND s.slot_date BETWEEN $3 AND $4
		ORDER BY s.slot_date, s.start_time`

	rows, err := r.db.Query(ctx, query, turfID, ownerProfileID, from, to)
	if err != nil {
		if isInvalidUUID(err) {
			return nil, ErrTurfNotFound
		}
		return nil, fmt.Errorf("select owner slots: %w", err)
	}
	defer rows.Close()

	slots, err := scanSlots(rows)
	if err != nil {
		return nil, err
	}
	if _, err := r.TurfByOwnerAndID(ctx, ownerProfileID, turfID); err != nil {
		return nil, err
	}
	return slots, nil
}

// PublicSlotsForDate lists one APPROVED turf's slots for a single date, for
// anyone to browse. A turf in any other status reports ErrTurfNotFound, the
// same as PublicTurfByID, so a guessed id cannot be used to find availability
// before the turf is approved.
func (r *Repository) PublicSlotsForDate(ctx context.Context, turfID string, date time.Time) ([]Slot, error) {
	const query = `
		SELECT ` + slotColumns + `
		FROM turf_slots s
		JOIN turfs t ON t.id = s.turf_id
		WHERE s.turf_id = $1 AND t.status = 'APPROVED' AND s.slot_date = $2
		ORDER BY s.start_time`

	rows, err := r.db.Query(ctx, query, turfID, date)
	if err != nil {
		if isInvalidUUID(err) {
			return nil, ErrTurfNotFound
		}
		return nil, fmt.Errorf("select public slots: %w", err)
	}
	defer rows.Close()

	slots, err := scanSlots(rows)
	if err != nil {
		return nil, err
	}
	if _, err := r.PublicTurfByID(ctx, turfID); err != nil {
		return nil, err
	}
	return slots, nil
}

// slotCandidate is one slot InsertSlots will try to create. It carries no
// owner or turf key: InsertSlots takes those once, for the whole batch.
type slotCandidate struct {
	Date      time.Time
	StartTime string
	EndTime   string
	Price     float64
}

// InsertSlots bulk-inserts generated slots in one round trip. Each insert is
// independently scoped to the owner's turf, the same defense-in-depth every
// other write in this package uses, and skips a slot that already exists
// (ON CONFLICT DO NOTHING backed by the turf_slots_no_overlap exclusion
// constraint) rather than erroring, so a repeat generate call over the same
// range is safe to retry.
func (r *Repository) InsertSlots(ctx context.Context, ownerProfileID, turfID string, candidates []slotCandidate) error {
	if len(candidates) == 0 {
		return nil
	}

	const insert = `
		INSERT INTO turf_slots (turf_id, slot_date, start_time, end_time, price)
		SELECT t.id, $3, $4, $5, $6
		FROM turfs t
		WHERE t.id = $1 AND t.owner_id = $2
		ON CONFLICT DO NOTHING`

	batch := &pgx.Batch{}
	for _, c := range candidates {
		batch.Queue(insert, turfID, ownerProfileID, c.Date, c.StartTime, c.EndTime, c.Price)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range candidates {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert slot: %w", err)
		}
	}
	return nil
}

// SetSlotStatus toggles one of an owner's own slots between OPEN and
// BLOCKED.
func (r *Repository) SetSlotStatus(ctx context.Context, ownerProfileID, turfID, slotID string, status SlotStatus) (Slot, error) {
	const update = `
		UPDATE turf_slots SET status = $4
		FROM turfs t
		WHERE turf_slots.turf_id = t.id AND t.id = $2 AND t.owner_id = $1 AND turf_slots.id = $3`

	tag, err := r.db.Exec(ctx, update, ownerProfileID, turfID, slotID, status)
	if err != nil {
		if isInvalidUUID(err) {
			return Slot{}, ErrSlotNotFound
		}
		return Slot{}, fmt.Errorf("update slot status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Slot{}, ErrSlotNotFound
	}
	return r.slotByID(ctx, ownerProfileID, turfID, slotID)
}

// DeleteSlot removes one of an owner's own generated slots.
func (r *Repository) DeleteSlot(ctx context.Context, ownerProfileID, turfID, slotID string) error {
	const query = `
		DELETE FROM turf_slots USING turfs t
		WHERE turf_slots.turf_id = t.id AND t.id = $2 AND t.owner_id = $1 AND turf_slots.id = $3`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, slotID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrSlotNotFound
		}
		return fmt.Errorf("delete slot: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSlotNotFound
	}
	return nil
}

func (r *Repository) slotByID(ctx context.Context, ownerProfileID, turfID, slotID string) (Slot, error) {
	const query = `
		SELECT ` + slotColumns + `
		FROM turf_slots s
		JOIN turfs t ON t.id = s.turf_id
		WHERE s.turf_id = $1 AND t.owner_id = $2 AND s.id = $3`

	row, err := scanSlotRow(r.db.QueryRow(ctx, query, turfID, ownerProfileID, slotID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Slot{}, ErrSlotNotFound
		}
		return Slot{}, fmt.Errorf("select slot: %w", err)
	}
	return row.toSlot(), nil
}

// --- blocked dates -----------------------------------------------------------

// BlockedDates lists every blocked date for one of an owner's own turfs.
func (r *Repository) BlockedDates(ctx context.Context, ownerProfileID, turfID string) ([]BlockedDate, error) {
	const query = `
		SELECT bd.id::text, bd.blocked_date, coalesce(bd.reason, '')
		FROM turf_blocked_dates bd
		JOIN turfs t ON t.id = bd.turf_id
		WHERE bd.turf_id = $1 AND t.owner_id = $2
		ORDER BY bd.blocked_date`

	rows, err := r.db.Query(ctx, query, turfID, ownerProfileID)
	if err != nil {
		if isInvalidUUID(err) {
			return nil, ErrTurfNotFound
		}
		return nil, fmt.Errorf("select blocked dates: %w", err)
	}
	defer rows.Close()

	out := make([]BlockedDate, 0, 8)
	for rows.Next() {
		var row blockedDateRow
		if err := rows.Scan(&row.ID, &row.Date, &row.Reason); err != nil {
			return nil, fmt.Errorf("scan blocked date: %w", err)
		}
		out = append(out, row.toBlockedDate())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocked dates: %w", err)
	}
	if _, err := r.TurfByOwnerAndID(ctx, ownerProfileID, turfID); err != nil {
		return nil, err
	}
	return out, nil
}

// BlockDate blocks a whole day on one of an owner's own turfs.
func (r *Repository) BlockDate(ctx context.Context, ownerProfileID, turfID string, date time.Time, reason string) (BlockedDate, error) {
	const insert = `
		INSERT INTO turf_blocked_dates (turf_id, blocked_date, reason)
		SELECT t.id, $3, nullif($4, '')
		FROM turfs t
		WHERE t.id = $1 AND t.owner_id = $2
		RETURNING id::text, blocked_date, coalesce(reason, '')`

	var row blockedDateRow
	err := r.db.QueryRow(ctx, insert, turfID, ownerProfileID, date, reason).Scan(&row.ID, &row.Date, &row.Reason)
	if err != nil {
		if isUniqueViolation(err) {
			return BlockedDate{}, ErrDateAlreadyBlocked
		}
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return BlockedDate{}, ErrTurfNotFound
		}
		return BlockedDate{}, fmt.Errorf("insert blocked date: %w", err)
	}
	return row.toBlockedDate(), nil
}

// UnblockDate removes a blocked-date row from one of an owner's own turfs.
func (r *Repository) UnblockDate(ctx context.Context, ownerProfileID, turfID, blockedDateID string) error {
	const query = `
		DELETE FROM turf_blocked_dates USING turfs t
		WHERE turf_blocked_dates.turf_id = t.id AND t.id = $2 AND t.owner_id = $1 AND turf_blocked_dates.id = $3`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, blockedDateID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrBlockedDateNotFound
		}
		return fmt.Errorf("delete blocked date: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrBlockedDateNotFound
	}
	return nil
}

// --- blocked time ranges -----------------------------------------------------

// BlockedTimeRanges lists every blocked time range for one of an owner's own
// turfs.
func (r *Repository) BlockedTimeRanges(ctx context.Context, ownerProfileID, turfID string) ([]BlockedTimeRange, error) {
	const query = `
		SELECT btr.id::text, btr.blocked_date, btr.start_time, btr.end_time, coalesce(btr.reason, '')
		FROM turf_blocked_time_ranges btr
		JOIN turfs t ON t.id = btr.turf_id
		WHERE btr.turf_id = $1 AND t.owner_id = $2
		ORDER BY btr.blocked_date, btr.start_time`

	rows, err := r.db.Query(ctx, query, turfID, ownerProfileID)
	if err != nil {
		if isInvalidUUID(err) {
			return nil, ErrTurfNotFound
		}
		return nil, fmt.Errorf("select blocked time ranges: %w", err)
	}
	defer rows.Close()

	out := make([]BlockedTimeRange, 0, 8)
	for rows.Next() {
		var row blockedTimeRangeRow
		if err := rows.Scan(&row.ID, &row.Date, &row.StartTime, &row.EndTime, &row.Reason); err != nil {
			return nil, fmt.Errorf("scan blocked time range: %w", err)
		}
		out = append(out, row.toBlockedTimeRange())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocked time ranges: %w", err)
	}
	if _, err := r.TurfByOwnerAndID(ctx, ownerProfileID, turfID); err != nil {
		return nil, err
	}
	return out, nil
}

// BlockTimeRange blocks part of one day on one of an owner's own turfs. Two
// ranges that overlap on the same turf and date are rejected by the
// turf_blocked_time_ranges_no_overlap exclusion constraint, the same
// structural guard turf_slots uses.
func (r *Repository) BlockTimeRange(ctx context.Context, ownerProfileID, turfID string, date time.Time, startTime, endTime, reason string) (BlockedTimeRange, error) {
	const insert = `
		INSERT INTO turf_blocked_time_ranges (turf_id, blocked_date, start_time, end_time, reason)
		SELECT t.id, $3, $4, $5, nullif($6, '')
		FROM turfs t
		WHERE t.id = $1 AND t.owner_id = $2
		RETURNING id::text, blocked_date, start_time, end_time, coalesce(reason, '')`

	var row blockedTimeRangeRow
	err := r.db.QueryRow(ctx, insert, turfID, ownerProfileID, date, startTime, endTime, reason).
		Scan(&row.ID, &row.Date, &row.StartTime, &row.EndTime, &row.Reason)
	if err != nil {
		if isExclusionViolation(err) {
			return BlockedTimeRange{}, ErrTimeRangeOverlapsBlock
		}
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return BlockedTimeRange{}, ErrTurfNotFound
		}
		return BlockedTimeRange{}, fmt.Errorf("insert blocked time range: %w", err)
	}
	return row.toBlockedTimeRange(), nil
}

// UnblockTimeRange removes a blocked-time-range row from one of an owner's
// own turfs.
func (r *Repository) UnblockTimeRange(ctx context.Context, ownerProfileID, turfID, blockedTimeRangeID string) error {
	const query = `
		DELETE FROM turf_blocked_time_ranges USING turfs t
		WHERE turf_blocked_time_ranges.turf_id = t.id AND t.id = $2 AND t.owner_id = $1 AND turf_blocked_time_ranges.id = $3`

	tag, err := r.db.Exec(ctx, query, ownerProfileID, turfID, blockedTimeRangeID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrBlockedTimeRangeNotFound
		}
		return fmt.Errorf("delete blocked time range: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrBlockedTimeRangeNotFound
	}
	return nil
}

func scanSlots(rows pgx.Rows) ([]Slot, error) {
	out := make([]Slot, 0, 32)
	for rows.Next() {
		row, err := scanSlotRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan slot: %w", err)
		}
		out = append(out, row.toSlot())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate slots: %w", err)
	}
	return out, nil
}

func scanSlotRow(row pgx.Row) (slotRow, error) {
	var s slotRow
	err := row.Scan(&s.ID, &s.Date, &s.StartTime, &s.EndTime, &s.Price, &s.Status, &s.Available, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}
