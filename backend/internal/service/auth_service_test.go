package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/token"
)

var testJWTSecret = []byte("test-secret-key-not-for-production")

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func TestLogin_ValidCredentials_ReturnsTokenPair(t *testing.T) {
	// Arrange
	userID := uuid.New()
	users := &mockUserRepository{
		getByEmailFunc: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: userID, Email: email, PasswordHash: hashPassword(t, "correct-password")}, nil
		},
	}
	authService := NewAuthService(users, testJWTSecret, time.Hour, time.Hour*24)

	// Act
	pair, err := authService.Login(context.Background(), "owner@brewops.mx", "correct-password")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected both tokens to be non-empty")
	}
}

func TestLogin_WrongPassword_ReturnsInvalidCredentials(t *testing.T) {
	// Arrange
	users := &mockUserRepository{
		getByEmailFunc: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: uuid.New(), Email: email, PasswordHash: hashPassword(t, "correct-password")}, nil
		},
	}
	authService := NewAuthService(users, testJWTSecret, time.Hour, time.Hour*24)

	// Act
	_, err := authService.Login(context.Background(), "owner@brewops.mx", "wrong-password")

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUser_ReturnsInvalidCredentials(t *testing.T) {
	// Arrange
	users := &mockUserRepository{
		getByEmailFunc: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}
	authService := NewAuthService(users, testJWTSecret, time.Hour, time.Hour*24)

	// Act
	_, err := authService.Login(context.Background(), "nobody@brewops.mx", "whatever")

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRefresh_ValidToken_ReturnsNewAccessToken(t *testing.T) {
	// Arrange
	authService := NewAuthService(&mockUserRepository{}, testJWTSecret, time.Hour, time.Hour*24)
	refreshToken, err := token.Generate(testJWTSecret, uuid.New(), token.KindRefresh, time.Hour*24*30)
	if err != nil {
		t.Fatalf("failed to generate refresh token fixture: %v", err)
	}

	// Act
	access, err := authService.Refresh(context.Background(), refreshToken)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if access == "" {
		t.Fatal("expected a non-empty access token")
	}
}

func TestRegister_ValidInput_CreatesAdminUser(t *testing.T) {
	// Arrange
	var created domain.User
	users := &mockUserRepository{
		createFunc: func(_ context.Context, user *domain.User) error {
			user.ID = uuid.New()
			created = *user
			return nil
		},
	}
	authService := NewAuthService(users, testJWTSecret, time.Hour, time.Hour*24)

	// Act
	user, err := authService.Register(context.Background(), "owner@brewops.mx", "a-strong-password")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != "owner@brewops.mx" || user.Role != "ADMIN" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if created.PasswordHash == "" || created.PasswordHash == "a-strong-password" {
		t.Fatal("expected password to be hashed before Create is called")
	}
}

func TestRefresh_ExpiredToken_ReturnsInvalidRefreshToken(t *testing.T) {
	// Arrange
	authService := NewAuthService(&mockUserRepository{}, testJWTSecret, time.Hour, time.Hour*24)
	expiredToken, err := token.Generate(testJWTSecret, uuid.New(), token.KindRefresh, -time.Hour)
	if err != nil {
		t.Fatalf("failed to generate expired token fixture: %v", err)
	}

	// Act
	_, err = authService.Refresh(context.Background(), expiredToken)

	// Assert
	if !errors.Is(err, domain.ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}
}
