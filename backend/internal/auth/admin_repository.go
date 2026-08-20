package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Users lists accounts for the admin surface, newest first. limit and offset
// are expected to already be validated by the caller (the service clamps
// them); the repository just runs the query.
func (r *Repository) Users(ctx context.Context, limit, offset int) ([]User, error) {
	const query = `SELECT ` + userColumns + ` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0, limit)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

// UserCount returns the total number of accounts, for pagination metadata.
func (r *Repository) UserCount(ctx context.Context) (int, error) {
	const query = `SELECT count(*) FROM users`

	var total int
	if err := r.db.QueryRow(ctx, query).Scan(&total); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return total, nil
}

// SetActive flips an account's is_active flag and returns the updated row.
// Setting the flag to its current value is not an error: it is idempotent for
// the same reason logout is, so a client retrying a deactivate call is not
// told its own action failed.
func (r *Repository) SetActive(ctx context.Context, userID string, active bool) (User, error) {
	const query = `
		UPDATE users SET is_active = $2
		WHERE id = $1
		RETURNING ` + userColumns

	user, err := scanUser(r.db.QueryRow(ctx, query, userID, active))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("update user active flag: %w", err)
	}
	return user, nil
}
