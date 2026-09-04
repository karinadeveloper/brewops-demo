package main

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/config"
	"github.com/kariaranelly/brew-ops/backend/internal/storage"
)

// unreachableDSN is syntactically valid but never actually contacted:
// pgxpool.New only parses the connection string and builds a lazy pool, it
// never dials until something first acquires a connection — neither of the
// two checklist behaviors under test here (route-not-registered, and a
// rejected reset token) ever reaches that point.
const unreachableDSN = "postgres://user:pass@127.0.0.1:1/brewops?sslmode=disable"

func newTestApp(t *testing.T, cfg *config.Config) *fiber.App {
	t.Helper()

	pool, err := pgxpool.New(context.Background(), unreachableDSN)
	if err != nil {
		t.Fatalf("build test pool: %v", err)
	}
	t.Cleanup(pool.Close)

	storageClient, err := storage.NewLocalDiskStorageClient(t.TempDir(), cfg.PublicBaseURL)
	if err != nil {
		t.Fatalf("build test storage client: %v", err)
	}

	app := fiber.New()
	registerRoutes(app, cfg, pool, storageClient)
	return app
}

func baseTestConfig() *config.Config {
	return &config.Config{
		JWTSecret:          "test-secret",
		JWTAccessTokenTTL:  0,
		JWTRefreshTokenTTL: 0,
		PublicBaseURL:      "http://localhost:8080",
	}
}

func TestRegisterRoutes_DemoModeFalse_ResetEndpointNotRegistered(t *testing.T) {
	// Arrange
	cfg := baseTestConfig()
	cfg.DemoMode = false
	app := newTestApp(t, cfg)
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/demo-reset", nil)

	// Act
	resp, err := app.Test(req)

	// Assert — a plain unmatched-route 404, not a handler that reveals the
	// route exists.
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected 404 when DEMO_MODE=false, got %d", resp.StatusCode)
	}
}

func TestRegisterRoutes_DemoModeTrue_MissingToken_Rejected(t *testing.T) {
	// Arrange
	cfg := baseTestConfig()
	cfg.DemoMode = true
	cfg.DemoResetToken = "correct-token"
	app := newTestApp(t, cfg)
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/demo-reset", nil)

	// Act
	resp, err := app.Test(req)

	// Assert — route exists (not a 404) but the reset never runs without a
	// valid shared secret.
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode == fiber.StatusNotFound {
		t.Fatal("expected the route to be registered when DEMO_MODE=true")
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 with a missing token, got %d", resp.StatusCode)
	}
}

func TestRegisterRoutes_DemoModeTrue_WrongToken_Rejected(t *testing.T) {
	// Arrange
	cfg := baseTestConfig()
	cfg.DemoMode = true
	cfg.DemoResetToken = "correct-token"
	app := newTestApp(t, cfg)
	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/admin/demo-reset", nil)
	req.Header.Set("X-Demo-Reset-Token", "wrong-token")

	// Act
	resp, err := app.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 with a wrong token, got %d", resp.StatusCode)
	}
}
