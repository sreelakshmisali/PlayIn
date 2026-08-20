package auth

import "context"

// Pagination bounds for the admin user list. The default keeps a page small
// enough to render comfortably; the ceiling stops an unbounded query.
const (
	defaultUserPageSize = 20
	maxUserPageSize     = 100
)

// UserPage is one page of the admin account listing.
type UserPage struct {
	Users  []Profile `json:"users"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

// ListUsers returns a page of accounts. limit and offset are clamped here
// rather than trusted from the caller: a page size request of 0 or 10000 is
// an HTTP-layer input, and deciding what to do with an unreasonable one is a
// business rule, not a query concern.
func (s *Service) ListUsers(ctx context.Context, limit, offset int) (UserPage, error) {
	switch {
	case limit <= 0:
		limit = defaultUserPageSize
	case limit > maxUserPageSize:
		limit = maxUserPageSize
	}
	if offset < 0 {
		offset = 0
	}

	users, err := s.store.Users(ctx, limit, offset)
	if err != nil {
		return UserPage{}, err
	}
	total, err := s.store.UserCount(ctx)
	if err != nil {
		return UserPage{}, err
	}

	profiles := make([]Profile, 0, len(users))
	for _, u := range users {
		profiles = append(profiles, u.Profile())
	}

	return UserPage{Users: profiles, Total: total, Limit: limit, Offset: offset}, nil
}

// AdminUser reads one account for the admin surface.
func (s *Service) AdminUser(ctx context.Context, userID string) (Profile, error) {
	user, err := s.store.UserByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	return user.Profile(), nil
}

// SetUserActive activates or deactivates an account.
//
// callerID is the admin performing the action; if it names the same account
// as userID, the action is refused. IsActive is re-checked on every request
// (see Authenticate), so an admin who deactivated themselves would be locked
// out immediately, including from the endpoint that could undo it.
func (s *Service) SetUserActive(ctx context.Context, callerID, userID string, active bool) (Profile, error) {
	if callerID == userID {
		return Profile{}, ErrCannotModifySelf
	}

	user, err := s.store.SetActive(ctx, userID, active)
	if err != nil {
		return Profile{}, err
	}
	return user.Profile(), nil
}
