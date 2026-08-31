//go:build integration

// Package demoseed_test exercises Reset against a real Postgres database —
// the only way to prove the delete-then-reinsert cycle is actually
// idempotent (no accumulating rows, no FK violations in delete order).
// Run with:
//
//	go test -tags=integration ./internal/demoseed/...
//
// Requires a reachable Postgres at DATABASE_URL with migrations applied:
//
//	docker-compose up -d postgres
//	migrate -path db/migrations -database "$DATABASE_URL" up
package demoseed_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/demoseed"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestReset_RunTwice_LeavesEquivalentStateWithNoDuplicates(t *testing.T) {
	// Arrange
	pool := testPool(t)
	dir := t.TempDir()
	opts := demoseed.Options{
		AdminEmail:      "demo-integration-test@brewops.mx",
		AdminPassword:   "Demo2026!",
		LocalStorageDir: filepath.Join(dir, "local-storage"),
		PublicBaseURL:   "http://localhost:8080",
	}

	// Act
	first, err := demoseed.Reset(context.Background(), pool, opts)
	if err != nil {
		t.Fatalf("first reset: %v", err)
	}
	second, err := demoseed.Reset(context.Background(), pool, opts)
	if err != nil {
		t.Fatalf("second reset: %v", err)
	}

	// Assert
	if first != second {
		t.Fatalf("expected identical summaries across runs, got %+v and %+v", first, second)
	}

	var productCount, saleCount, adminCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM products").Scan(&productCount); err != nil {
		t.Fatalf("count products: %v", err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM sales").Scan(&saleCount); err != nil {
		t.Fatalf("count sales: %v", err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM users WHERE email = $1", opts.AdminEmail).Scan(&adminCount); err != nil {
		t.Fatalf("count admin users: %v", err)
	}

	if productCount != second.Products {
		t.Errorf("expected %d products in the database, got %d", second.Products, productCount)
	}
	if saleCount != second.Sales {
		t.Errorf("expected %d sales in the database, got %d", second.Sales, saleCount)
	}
	if adminCount != 1 {
		t.Errorf("expected exactly one demo admin user across both resets, got %d", adminCount)
	}
}

func TestReset_MissingAdminCredentials_ReturnsError(t *testing.T) {
	// Arrange
	pool := testPool(t)

	// Act
	_, err := demoseed.Reset(context.Background(), pool, demoseed.Options{})

	// Assert
	if err == nil {
		t.Fatal("expected an error when admin credentials are missing")
	}
}
