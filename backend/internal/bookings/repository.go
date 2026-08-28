package bookings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/orgmelethil/playhub/backend/internal/database"
)

// invalidTextRepresentation is PostgreSQL's SQLSTATE for a malformed literal,
// which is what a non-UUID path parameter produces.
const invalidTextRepresentation = "22P02"

// uniqueViolation is PostgreSQL's SQLSTATE for a unique constraint breach —
// here, bookings_slot_confirmed_key, the partial unique index that is the
// structural backstop behind CreateBooking's guarded reservation.
const uniqueViolation = "23505"

// Repository is the only place in the package that writes SQL.
type Repository struct {
	db *database.Pool
}

// NewRepository wires a Repository over the shared connection pool.
func NewRepository(db *database.Pool) *Repository {
	return &Repository{db: db}
}

// bookingColumns is the projection every booking read shares. Date,
// start_time and end_time are joined in from the reserved slot; everything
// else comes from bookings itself.
const bookingColumns = `
	b.id::text, b.player_id::text, b.turf_id::text, b.turf_slot_id::text,
	ts.slot_date, ts.start_time, ts.end_time,
	b.status, b.price, b.created_at, b.updated_at, b.cancelled_at`

// CreateBooking reserves a slot and records the booking in one transaction.
//
// The reservation is the guarded UPDATE turf_slots ... WHERE status = 'OPEN'
// shape the owners package already uses for every slot status transition
// (see migration 000006's own comment on turf_slots.status), extended with
// exactly the two conditions turf_slots' own read projection uses to compute
// "available": the slot's turf must be APPROVED, and the slot must fall
// inside neither a blocked date nor a blocked time range. PostgreSQL holds a
// row lock on the slot for the statement's duration, so a second, concurrent
// call targeting the same slot blocks until the first commits, then
// re-evaluates the WHERE clause against the now-committed status = 'BOOKED'
// row and matches nothing. No SELECT ... FOR UPDATE or advisory lock is
// needed.
//
// The insert that follows is additionally guarded by
// bookings_slot_confirmed_key, a partial unique index allowing at most one
// CONFIRMED booking per slot. That index is what would still refuse a second
// booking even if the reservation UPDATE above were ever bypassed or
// miswritten — a second, independent guarantee, not the primary mechanism.
func (r *Repository) CreateBooking(ctx context.Context, playerID, slotID string) (Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Booking{}, fmt.Errorf("begin booking transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once the transaction is committed

	const reserve = `
		UPDATE turf_slots ts
		SET status = 'BOOKED'
		FROM turfs t
		WHERE ts.id = $1
		  AND ts.turf_id = t.id
		  AND t.status = 'APPROVED'
		  AND ts.status = 'OPEN'
		  AND NOT EXISTS (
			  SELECT 1 FROM turf_blocked_dates bd
			  WHERE bd.turf_id = ts.turf_id AND bd.blocked_date = ts.slot_date
		  )
		  AND NOT EXISTS (
			  SELECT 1 FROM turf_blocked_time_ranges btr
			  WHERE btr.turf_id = ts.turf_id AND btr.blocked_date = ts.slot_date
				AND btr.minute_range && ts.minute_range
		  )
		RETURNING ts.turf_id::text, ts.price, ts.slot_date, ts.start_time, ts.end_time`

	var turfID string
	var price float64
	var date time.Time
	var startTime, endTime string

	err = tx.QueryRow(ctx, reserve, slotID).Scan(&turfID, &price, &date, &startTime, &endTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Booking{}, ErrSlotNotBookable
		}
		return Booking{}, fmt.Errorf("reserve slot: %w", err)
	}

	const insert = `
		INSERT INTO bookings (player_id, turf_id, turf_slot_id, price)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, status, created_at, updated_at, cancelled_at`

	row := bookingRow{
		PlayerID: playerID, TurfID: turfID, TurfSlotID: slotID,
		Date: date, StartTime: startTime, EndTime: endTime,
		Price: price,
	}

	err = tx.QueryRow(ctx, insert, playerID, turfID, slotID, price).
		Scan(&row.ID, &row.Status, &row.CreatedAt, &row.UpdatedAt, &row.CancelledAt)
	if err != nil {
		if isUniqueViolation(err) {
			return Booking{}, ErrSlotNotBookable
		}
		return Booking{}, fmt.Errorf("insert booking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Booking{}, fmt.Errorf("commit booking transaction: %w", err)
	}
	return row.toBooking(), nil
}

// BookingsForPlayer lists one player's own bookings, most recent first.
func (r *Repository) BookingsForPlayer(ctx context.Context, playerID string) ([]Booking, error) {
	const query = `
		SELECT ` + bookingColumns + `
		FROM bookings b
		JOIN turf_slots ts ON ts.id = b.turf_slot_id
		WHERE b.player_id = $1
		ORDER BY b.created_at DESC`

	rows, err := r.db.Query(ctx, query, playerID)
	if err != nil {
		if isInvalidUUID(err) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("select bookings: %w", err)
	}
	defer rows.Close()

	out := make([]Booking, 0, 16)
	for rows.Next() {
		row, err := scanBookingRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		out = append(out, row.toBooking())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bookings: %w", err)
	}
	return out, nil
}

// BookingByID reads one of a player's own bookings. A booking that does not
// exist, or belongs to a different player, reports ErrBookingNotFound —
// identically, so a guessed id cannot be used to tell the two apart.
func (r *Repository) BookingByID(ctx context.Context, playerID, bookingID string) (Booking, error) {
	const query = `
		SELECT ` + bookingColumns + `
		FROM bookings b
		JOIN turf_slots ts ON ts.id = b.turf_slot_id
		WHERE b.player_id = $1 AND b.id = $2`

	row, err := scanBookingRow(r.db.QueryRow(ctx, query, playerID, bookingID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Booking{}, ErrBookingNotFound
		}
		return Booking{}, fmt.Errorf("select booking: %w", err)
	}
	return row.toBooking(), nil
}

// CancelBooking moves one of a player's own bookings from CONFIRMED to
// CANCELLED and releases its slot back to OPEN, in one transaction.
//
// A booking that does not exist, or belongs to a different player, reports
// ErrBookingNotFound. One that exists but is already CANCELLED reports
// ErrAlreadyCancelled instead — the same "guarded update, then look again
// only on a miss" shape turfs.SubmitTurf already uses to tell "not found"
// apart from "wrong state to do this from".
func (r *Repository) CancelBooking(ctx context.Context, playerID, bookingID string) (Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Booking{}, fmt.Errorf("begin cancel transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once the transaction is committed

	const cancel = `
		UPDATE bookings SET status = 'CANCELLED', cancelled_at = now()
		WHERE id = $1 AND player_id = $2 AND status = 'CONFIRMED'
		RETURNING turf_slot_id::text`

	var slotID string
	err = tx.QueryRow(ctx, cancel, bookingID, playerID).Scan(&slotID)
	switch {
	case err == nil:
		// fall through to release the slot below
	case errors.Is(err, pgx.ErrNoRows), isInvalidUUID(err):
		exists, existsErr := r.bookingExists(ctx, playerID, bookingID)
		if existsErr != nil {
			return Booking{}, existsErr
		}
		if exists {
			return Booking{}, ErrAlreadyCancelled
		}
		return Booking{}, ErrBookingNotFound
	default:
		return Booking{}, fmt.Errorf("cancel booking: %w", err)
	}

	const release = `UPDATE turf_slots SET status = 'OPEN' WHERE id = $1 AND status = 'BOOKED'`
	if _, err := tx.Exec(ctx, release, slotID); err != nil {
		return Booking{}, fmt.Errorf("release slot: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Booking{}, fmt.Errorf("commit cancel transaction: %w", err)
	}
	return r.BookingByID(ctx, playerID, bookingID)
}

// bookingExists reports whether a booking with this id exists for this
// player, regardless of status. A malformed id is not an error here, only a
// "no": the caller already knows it wants ErrBookingNotFound for that case.
func (r *Repository) bookingExists(ctx context.Context, playerID, bookingID string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM bookings WHERE id = $1 AND player_id = $2)`

	var exists bool
	if err := r.db.QueryRow(ctx, query, bookingID, playerID).Scan(&exists); err != nil {
		if isInvalidUUID(err) {
			return false, nil
		}
		return false, fmt.Errorf("check booking exists: %w", err)
	}
	return exists, nil
}

func scanBookingRow(row pgx.Row) (bookingRow, error) {
	var b bookingRow
	err := row.Scan(
		&b.ID, &b.PlayerID, &b.TurfID, &b.TurfSlotID,
		&b.Date, &b.StartTime, &b.EndTime,
		&b.Status, &b.Price, &b.CreatedAt, &b.UpdatedAt, &b.CancelledAt,
	)
	return b, err
}

// isInvalidUUID reports whether err is PostgreSQL rejecting a malformed UUID
// literal. The parameter always comes from a path segment or a request body,
// so a bad value means a bad request, not a server fault.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation
}

// isUniqueViolation reports whether err is PostgreSQL refusing a duplicate on
// a unique index.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}
