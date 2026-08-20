package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/orgmelethil/playhub/backend/internal/database"
)

// uniqueViolation is PostgreSQL's SQLSTATE for a unique constraint breach.
const uniqueViolation = "23505"

// Repository is the only place in the package that writes SQL.
//
// It reports storage facts and nothing more: a duplicate email comes back as
// ErrEmailTaken, a missing row as pgx.ErrNoRows. Deciding what either means is
// the service's job.
type Repository struct {
	db *database.Pool
}

// NewRepository wires a Repository over the shared connection pool.
func NewRepository(db *database.Pool) *Repository {
	return &Repository{db: db}
}

// userColumns is the projection every user read shares, so a column added to
// the scan is added in exactly one place.
const userColumns = `id::text, email, password_hash, full_name, role, is_active, created_at, updated_at`

// CreateUser inserts an account and returns the stored row.
// It returns ErrEmailTaken when the address is already registered.
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, fullName string, role Role) (User, error) {
	const query = `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + userColumns

	user, err := scanUser(r.db.QueryRow(ctx, query, email, passwordHash, fullName, string(role)))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	return user, nil
}

// UserByEmail looks an account up by its normalised address.
func (r *Repository) UserByEmail(ctx context.Context, email string) (User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE email = $1`

	user, err := scanUser(r.db.QueryRow(ctx, query, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("select user by email: %w", err)
	}
	return user, nil
}

// UserByID looks an account up by its UUID. An id that is not a valid UUID is
// reported as a missing user rather than a database error, because it can only
// come from a token this service signed against a row that no longer exists.
func (r *Repository) UserByID(ctx context.Context, id string) (User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	user, err := scanUser(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("select user by id: %w", err)
	}
	return user, nil
}

// CreateRefreshToken records an issued refresh token and returns its id, which
// becomes the token's jti claim.
func (r *Repository) CreateRefreshToken(ctx context.Context, userID string, expiresAt time.Time) (string, error) {
	const query = `
		INSERT INTO refresh_tokens (user_id, expires_at)
		VALUES ($1, $2)
		RETURNING id::text`

	var id string
	if err := r.db.QueryRow(ctx, query, userID, expiresAt).Scan(&id); err != nil {
		return "", fmt.Errorf("insert refresh token: %w", err)
	}
	return id, nil
}

// RefreshTokenByID reads a refresh token record. A missing row is reported as
// ErrInvalidToken: a correctly signed token whose record is gone is a token
// that was issued by this service and then deleted, which is not usable.
func (r *Repository) RefreshTokenByID(ctx context.Context, id string) (RefreshRecord, error) {
	const query = `
		SELECT id::text, user_id::text, expires_at, revoked_at
		FROM refresh_tokens
		WHERE id = $1`

	var rec RefreshRecord
	err := r.db.QueryRow(ctx, query, id).Scan(&rec.ID, &rec.UserID, &rec.ExpiresAt, &rec.RevokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return RefreshRecord{}, ErrInvalidToken
		}
		return RefreshRecord{}, fmt.Errorf("select refresh token: %w", err)
	}
	return rec, nil
}

// RevokeRefreshToken marks one token unusable. Revoking an already revoked or
// unknown token is not an error: logout has to be idempotent, because a client
// retrying it must not be told its own sign-out failed.
func (r *Repository) RevokeRefreshToken(ctx context.Context, id string) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL`

	if _, err := r.db.Exec(ctx, query, id); err != nil {
		if isInvalidUUID(err) {
			return nil
		}
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// isInvalidUUID reports whether err is PostgreSQL rejecting a malformed UUID
// literal. The parameter always comes from a token claim, so a bad value means
// a forged or corrupted token, not a server fault.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
