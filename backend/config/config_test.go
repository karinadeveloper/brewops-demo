package config

import "testing"

func TestLoad_MissingDatabaseURL_ReturnsError(t *testing.T) {
	// Arrange
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "test-secret")

	// Act
	_, err := Load()

	// Assert
	if err == nil {
		t.Fatal("expected an error when DATABASE_URL is missing, got nil")
	}
}

func TestLoad_AllRequiredVarsPresent_ReturnsConfig(t *testing.T) {
	// Arrange
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/brewops")
	t.Setenv("JWT_SECRET", "test-secret")

	// Act
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/brewops" {
		t.Errorf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("unexpected JWTSecret: %s", cfg.JWTSecret)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Port)
	}
}
