package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasherRoundTrip(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)

	hash, err := hasher.Hash("correct horse 7")
	if err != nil {
		t.Fatalf("Hash() returned error: %v", err)
	}
	if hash == "correct horse 7" {
		t.Fatal("Hash() returned the password unchanged")
	}
	if err := hasher.Verify(hash, "correct horse 7"); err != nil {
		t.Errorf("Verify() returned error: %v", err)
	}
}

func TestBcryptHasherRejectsWrongPassword(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)

	hash, err := hasher.Hash("correct horse 7")
	if err != nil {
		t.Fatalf("Hash() returned error: %v", err)
	}

	if err := hasher.Verify(hash, "wrong horse 7"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Verify() error = %v, want ErrInvalidCredentials", err)
	}
}

// Two hashes of one password must differ, or the salt is not doing its job and
// the table becomes vulnerable to precomputation.
func TestBcryptHasherSaltsEachHash(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)

	first, err := hasher.Hash("correct horse 7")
	if err != nil {
		t.Fatalf("Hash() returned error: %v", err)
	}
	second, err := hasher.Hash("correct horse 7")
	if err != nil {
		t.Fatalf("Hash() returned error: %v", err)
	}

	if first == second {
		t.Error("two hashes of the same password are identical, want different salts")
	}
}

func TestBcryptHasherRejectsOverlongPassword(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)

	if _, err := hasher.Hash(strings.Repeat("a", maxPasswordBytes+1)); err == nil {
		t.Error("Hash() returned nil error for a password past the 72 byte bcrypt limit")
	}
}

func TestBcryptHasherVerifyRejectsGarbageHash(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)

	err := hasher.Verify("not-a-bcrypt-hash", "correct horse 7")
	if err == nil {
		t.Fatal("Verify() returned nil error for a malformed hash")
	}
	// A malformed stored hash is a server fault, not a wrong password.
	if errors.Is(err, ErrInvalidCredentials) {
		t.Error("Verify() reported a malformed hash as invalid credentials")
	}
}
