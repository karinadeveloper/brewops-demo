package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/token"
)

// bcryptCost sits in the middle of the recommended 10-12 range.
const bcryptCost = 11

// TokenPair is the result of a successful login: an access token for
// authenticating requests and a refresh token for obtaining new ones.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// AuthService implements login, refresh, and registration against a
// domain.UserRepository. Token lifetimes default to 8h access / 30d
// refresh — a deliberately long-lived pair appropriate for a single-admin
// app, where the usual 15min/7day pattern would only add login friction.
type AuthService struct {
	users      domain.UserRepository
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(users domain.UserRepository, jwtSecret []byte, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		users:      users,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Login validates email/password and issues a new token pair. An unknown
// email and a wrong password both return domain.ErrInvalidCredentials —
// deliberately indistinguishable so the API never confirms which part was
// wrong.
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth: look up user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return s.issueTokenPair(user.ID)
}

// Refresh validates a refresh token and issues a new access token. It is
// stateless: it trusts a validly-signed, unexpired refresh token without
// re-checking the user still exists, since UserRepository (by design) only
// exposes GetByEmail and Create, not a by-ID lookup.
func (s *AuthService) Refresh(_ context.Context, refreshToken string) (string, error) {
	claims, err := token.Parse(s.jwtSecret, refreshToken, token.KindRefresh)
	if err != nil {
		return "", domain.ErrInvalidRefreshToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return "", domain.ErrInvalidRefreshToken
	}

	access, err := token.Generate(s.jwtSecret, userID, token.KindAccess, s.accessTTL)
	if err != nil {
		return "", fmt.Errorf("auth: generate access token: %w", err)
	}
	return access, nil
}

// Register creates the single ADMIN user. Callers (the handler) are
// responsible for only exposing this outside development mode.
func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Role:         "ADMIN",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) issueTokenPair(userID uuid.UUID) (*TokenPair, error) {
	access, err := token.Generate(s.jwtSecret, userID, token.KindAccess, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: generate access token: %w", err)
	}
	refresh, err := token.Generate(s.jwtSecret, userID, token.KindRefresh, s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: generate refresh token: %w", err)
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}
