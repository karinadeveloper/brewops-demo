package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrUserNotFound is returned when no active user matches the lookup.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when a Create call violates the
	// unique constraint on email.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrInvalidCredentials is returned for both an unknown email and a
	// wrong password, deliberately indistinguishable to avoid leaking
	// which one was wrong.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrInvalidRefreshToken is returned when a refresh token is expired,
	// malformed, or signed with the wrong key.
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// User is the single-admin account that can authenticate against the API.
// BrewOps has no multi-tenant concept — Role is always "ADMIN".
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	DeletedAt    *time.Time
	DeletedBy    *uuid.UUID
}

// UserRepository is the persistence boundary the auth service depends on.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
}
