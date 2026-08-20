package players

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

// Repository is the only place in the package that writes SQL.
type Repository struct {
	db *database.Pool
}

// NewRepository wires a Repository over the shared connection pool.
func NewRepository(db *database.Pool) *Repository {
	return &Repository{db: db}
}

// profileColumns is the projection every profile read shares.
//
// It selects from player_profiles only. Nothing from users is joined in, so
// there is no path by which an email address, a password hash, a role or an
// account flag can reach a caller through this package.
const profileColumns = `
	id::text, user_id::text,
	display_name, coalesce(image_url, ''), coalesce(bio, ''), coalesce(location, ''),
	created_at, updated_at`

// Sports returns the active catalogue, ordered by name so the client does not
// have to sort it.
func (r *Repository) Sports(ctx context.Context) ([]Sport, error) {
	const query = `
		SELECT id::text, slug, name, positions, created_at
		FROM sports
		WHERE is_active
		ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("select sports: %w", err)
	}
	defer rows.Close()

	sports := make([]Sport, 0, 8)
	for rows.Next() {
		var s Sport
		if err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.Positions, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan sport: %w", err)
		}
		sports = append(sports, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sports: %w", err)
	}

	return sports, nil
}

// SportByID reads one active sport. A retired sport reports ErrSportNotFound:
// it cannot be chosen, so for this package it does not exist.
func (r *Repository) SportByID(ctx context.Context, id string) (Sport, error) {
	const query = `
		SELECT id::text, slug, name, positions, created_at
		FROM sports
		WHERE id = $1 AND is_active`

	var s Sport
	err := r.db.QueryRow(ctx, query, id).Scan(&s.ID, &s.Slug, &s.Name, &s.Positions, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Sport{}, ErrSportNotFound
		}
		return Sport{}, fmt.Errorf("select sport: %w", err)
	}
	return s, nil
}

// ProfileByUserID reads a profile by the account that owns it.
func (r *Repository) ProfileByUserID(ctx context.Context, userID string) (Profile, error) {
	const query = `SELECT ` + profileColumns + ` FROM player_profiles WHERE user_id = $1`

	row, err := scanProfileRow(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Profile{}, ErrProfileNotFound
		}
		return Profile{}, fmt.Errorf("select player profile: %w", err)
	}

	sports, err := r.playerSports(ctx, row.ID)
	if err != nil {
		return Profile{}, err
	}
	return row.toProfile(sports), nil
}

// SaveProfile creates the profile or replaces its writable columns.
//
// The upsert is what makes one profile per account safe under concurrency: two
// simultaneous first-time saves both reach the unique index on user_id, and the
// loser updates instead of failing. The boolean reports whether a row was
// created, which the handler answers with 201 rather than 200.
func (r *Repository) SaveProfile(ctx context.Context, userID string, f profileFields) (Profile, bool, error) {
	const query = `
		INSERT INTO player_profiles (user_id, display_name, image_url, bio, location)
		VALUES ($1, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''))
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			image_url    = EXCLUDED.image_url,
			bio          = EXCLUDED.bio,
			location     = EXCLUDED.location
		RETURNING ` + profileColumns + `, (xmax = 0) AS inserted`

	var row profileRow
	var inserted bool

	err := r.db.QueryRow(ctx, query, userID, f.DisplayName, f.ImageURL, f.Bio, f.Location).Scan(
		&row.ID, &row.UserID,
		&row.DisplayName, &row.ImageURL, &row.Bio, &row.Location,
		&row.CreatedAt, &row.UpdatedAt,
		&inserted,
	)
	if err != nil {
		return Profile{}, false, fmt.Errorf("upsert player profile: %w", err)
	}

	sports, err := r.playerSports(ctx, row.ID)
	if err != nil {
		return Profile{}, false, err
	}
	return row.toProfile(sports), inserted, nil
}

// SetSport adds a preferred sport or changes the position on one already there.
//
// The insert selects from sports, so the sport must exist, be active, and offer
// the position before a row can be written. That check happens against the live
// row inside the same statement, not against a copy the service read earlier.
// Zero rows affected means one of those three conditions failed; the service
// works out which for the error message.
func (r *Repository) SetSport(ctx context.Context, profileID, sportID, position string) error {
	const query = `
		INSERT INTO player_sports (profile_id, sport_id, position)
		SELECT $1, s.id, nullif($3, '')
		FROM sports s
		WHERE s.id = $2
		  AND s.is_active
		  AND ($3 = '' OR $3 = ANY (s.positions))
		ON CONFLICT (profile_id, sport_id) DO UPDATE
			SET position = EXCLUDED.position`

	tag, err := r.db.Exec(ctx, query, profileID, sportID, position)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrSportNotFound
		}
		return fmt.Errorf("upsert player sport: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSportNotFound
	}
	return nil
}

// RemoveSport drops a preferred sport. Removing one the player does not have
// reports ErrSportNotPreferred rather than succeeding quietly, so a client
// deleting the wrong id finds out.
func (r *Repository) RemoveSport(ctx context.Context, profileID, sportID string) error {
	const query = `DELETE FROM player_sports WHERE profile_id = $1 AND sport_id = $2`

	tag, err := r.db.Exec(ctx, query, profileID, sportID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrSportNotPreferred
		}
		return fmt.Errorf("delete player sport: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSportNotPreferred
	}
	return nil
}

// ProfileIDForUser resolves the surrogate key the sports sub-resource needs.
// The profile's own id is never serialised, so this is the only way out of it.
func (r *Repository) ProfileIDForUser(ctx context.Context, userID string) (string, error) {
	const query = `SELECT id::text FROM player_profiles WHERE user_id = $1`

	var id string
	if err := r.db.QueryRow(ctx, query, userID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return "", ErrProfileNotFound
		}
		return "", fmt.Errorf("select profile id: %w", err)
	}
	return id, nil
}

// playerSports reads a profile's preferred sports with the sport joined in,
// ordered by name so the response is stable between reads.
func (r *Repository) playerSports(ctx context.Context, profileID string) ([]PlayerSport, error) {
	const query = `
		SELECT s.id::text, s.slug, s.name, s.positions, s.created_at, coalesce(ps.position, '')
		FROM player_sports ps
		JOIN sports s ON s.id = ps.sport_id
		WHERE ps.profile_id = $1
		ORDER BY s.name`

	rows, err := r.db.Query(ctx, query, profileID)
	if err != nil {
		return nil, fmt.Errorf("select player sports: %w", err)
	}
	defer rows.Close()

	out := make([]PlayerSport, 0, 4)
	for rows.Next() {
		var ps PlayerSport
		err := rows.Scan(&ps.Sport.ID, &ps.Sport.Slug, &ps.Sport.Name, &ps.Sport.Positions, &ps.Sport.CreatedAt, &ps.Position)
		if err != nil {
			return nil, fmt.Errorf("scan player sport: %w", err)
		}
		out = append(out, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate player sports: %w", err)
	}

	return out, nil
}

func scanProfileRow(row pgx.Row) (profileRow, error) {
	var p profileRow
	err := row.Scan(
		&p.ID, &p.UserID,
		&p.DisplayName, &p.ImageURL, &p.Bio, &p.Location,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

// isInvalidUUID reports whether err is PostgreSQL rejecting a malformed UUID
// literal. The parameter always comes from a path segment or a request body, so
// a bad value means a bad request, not a server fault.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation
}
