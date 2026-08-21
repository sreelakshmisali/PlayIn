package owners

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/orgmelethil/playhub/backend/internal/database"
)

// invalidTextRepresentation is PostgreSQL's SQLSTATE for a malformed literal,
// which is what a non-UUID path parameter produces.
const invalidTextRepresentation = "22P02"

// uniqueViolation is PostgreSQL's SQLSTATE for a unique constraint breach.
const uniqueViolation = "23505"

// exclusionViolation is PostgreSQL's SQLSTATE for an EXCLUDE constraint
// breach, the overlap guard turf_slots and turf_blocked_time_ranges use.
const exclusionViolation = "23P01"

// Repository is the only place in the package that writes SQL.
type Repository struct {
	db *database.Pool
}

// NewRepository wires a Repository over the shared connection pool.
func NewRepository(db *database.Pool) *Repository {
	return &Repository{db: db}
}

// profileColumns is the projection every owner profile read shares.
//
// It selects from owner_profiles only. Nothing from users is joined in, so
// there is no path by which an email address, a password hash, a role or an
// account flag can reach a caller through this package.
const profileColumns = `
	id::text, user_id::text,
	display_name, coalesce(phone, ''), coalesce(description, ''),
	created_at, updated_at`

// ProfileByUserID reads an owner profile by the account that owns it.
func (r *Repository) ProfileByUserID(ctx context.Context, userID string) (Profile, error) {
	const query = `SELECT ` + profileColumns + ` FROM owner_profiles WHERE user_id = $1`

	row, err := scanProfileRow(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Profile{}, ErrOwnerProfileNotFound
		}
		return Profile{}, fmt.Errorf("select owner profile: %w", err)
	}
	return row.toProfile(), nil
}

// SaveProfile creates the profile or replaces its writable columns.
//
// The upsert is what makes one profile per account safe under concurrency: two
// simultaneous first-time saves both reach the unique index on user_id, and the
// loser updates instead of failing. The boolean reports whether a row was
// created, which the handler answers with 201 rather than 200.
func (r *Repository) SaveProfile(ctx context.Context, userID string, f profileFields) (Profile, bool, error) {
	const query = `
		INSERT INTO owner_profiles (user_id, display_name, phone, description)
		VALUES ($1, $2, nullif($3, ''), nullif($4, ''))
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			phone        = EXCLUDED.phone,
			description  = EXCLUDED.description
		RETURNING ` + profileColumns + `, (xmax = 0) AS inserted`

	var row profileRow
	var inserted bool

	err := r.db.QueryRow(ctx, query, userID, f.DisplayName, f.Phone, f.Description).Scan(
		&row.ID, &row.UserID,
		&row.DisplayName, &row.Phone, &row.Description,
		&row.CreatedAt, &row.UpdatedAt,
		&inserted,
	)
	if err != nil {
		return Profile{}, false, fmt.Errorf("upsert owner profile: %w", err)
	}
	return row.toProfile(), inserted, nil
}

// ProfileIDForUser resolves the surrogate key turfs are anchored to. The
// profile's own id is never serialised, so this is the only way out of it.
func (r *Repository) ProfileIDForUser(ctx context.Context, userID string) (string, error) {
	const query = `SELECT id::text FROM owner_profiles WHERE user_id = $1`

	var id string
	if err := r.db.QueryRow(ctx, query, userID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return "", ErrOwnerProfileNotFound
		}
		return "", fmt.Errorf("select owner profile id: %w", err)
	}
	return id, nil
}

func scanProfileRow(row pgx.Row) (profileRow, error) {
	var p profileRow
	err := row.Scan(&p.ID, &p.UserID, &p.DisplayName, &p.Phone, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// isInvalidUUID reports whether err is PostgreSQL rejecting a malformed UUID
// literal. The parameter always comes from a path segment or a request body, so
// a bad value means a bad request, not a server fault.
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

// isExclusionViolation reports whether err is PostgreSQL refusing a row that
// overlaps another under an EXCLUDE constraint.
func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == exclusionViolation
}
