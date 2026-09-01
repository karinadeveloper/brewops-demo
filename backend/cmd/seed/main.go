// Command seed resets this demo deployment's database to a clean, realistic
// sample state: it deletes all existing business data and reinserts a
// sample product catalog, sales history, and marketing gallery, upserting
// the single demo admin account along the way. It is idempotent — running
// it again simply leaves the database in the same clean state, with no
// accumulating rows.
//
// This command exists only for this demo repo — see CLAUDE.md's "DEMO
// MODE" section. It shares its reset logic with POST
// /api/v1/admin/demo-reset via internal/demoseed.Reset; neither duplicates
// the other's logic.
//
// Usage:
//
//	go run ./cmd/seed
package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/kariaranelly/brew-ops/backend/config"
	"github.com/kariaranelly/brew-ops/backend/internal/demoseed"
)

// localStorageDir mirrors cmd/server/main.go's own constant — kept as a
// separate literal rather than an import to avoid pulling the server
// package's dependency graph into this small CLI.
const localStorageDir = "./local-storage"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// godotenv only loads .env for local development; in a scheduled
	// Cloud Run job the real environment variables are injected and this
	// is a silent no-op — mirrors cmd/server/main.go.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	summary, err := demoseed.Reset(ctx, pool, demoseed.Options{
		AdminEmail:      cfg.DemoAdminEmail,
		AdminPassword:   cfg.DemoAdminPassword,
		SeedAssetsDir:   filepath.Clean(cfg.SeedAssetsDir),
		LocalStorageDir: filepath.Clean(localStorageDir),
		PublicBaseURL:   cfg.PublicBaseURL,
	})
	if err != nil {
		return err
	}

	log.Printf(
		"seed complete: %d products, %d sales, %d marketing assets",
		summary.Products, summary.Sales, summary.MarketingAssets,
	)
	return nil
}
