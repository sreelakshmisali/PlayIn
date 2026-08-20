package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Hasher hashes and verifies passwords. The service depends on this interface
// rather than bcrypt directly, so tests can swap in a cheap implementation and
// not pay the work factor on every case.
type Hasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) error
}

// BcryptHasher is the production Hasher.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher wires a hasher at the configured work factor.
func NewBcryptHasher(cost int) *BcryptHasher {
	return &BcryptHasher{cost: cost}
}

// Hash returns the bcrypt hash of password.
//
// bcrypt silently truncates at 72 bytes, which would make two long passwords
// sharing a prefix interchangeable. Length is capped in validation, and this
// check is the backstop in case a caller bypasses it.
func (h *BcryptHasher) Hash(password string) (string, error) {
	if len(password) > maxPasswordBytes {
		return "", fmt.Errorf("hash password: longer than %d bytes", maxPasswordBytes)
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(digest), nil
}

// Verify reports whether password produced hash. It returns
// ErrInvalidCredentials for a mismatch so callers never branch on bcrypt's own
// error values.
func (h *BcryptHasher) Verify(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	switch {
	case err == nil:
		return nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return ErrInvalidCredentials
	default:
		return fmt.Errorf("verify password: %w", err)
	}
}
