// Package demoseed builds and tears down the realistic sample data the
// public demo deployment of this repo resets to periodically: products,
// inventory movements, sales history, and a marketing asset gallery. It
// never touches the users table except to upsert the single demo admin
// account, so that account survives every reset instead of being recreated.
//
// Reset is the one place this logic lives — both cmd/seed (a one-off CLI
// invocation, for local setup) and POST /api/v1/admin/demo-reset (via
// service.DemoService, for the scheduled reset in production) call it
// rather than duplicating the logic in two places.
//
// Demo-only: nothing in this package exists in the real BrewOps product —
// see CLAUDE.md's "DEMO MODE" section.
package demoseed

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Options configures one Reset call.
type Options struct {
	AdminEmail    string
	AdminPassword string
	// SeedAssetsDir holds the source images copied into LocalStorageDir on
	// every reset — see backend/seed-assets/. Optional: if empty, no images
	// are copied and every product/marketing asset gets a nil image_url.
	SeedAssetsDir string
	// LocalStorageDir is where seed images are copied to, mirroring
	// storage.LocalDiskStorageClient's own directory, and PublicBaseURL is
	// prefixed onto each copied filename to build the same kind of URL that
	// client would return for a real upload. The reset always writes
	// through this local-disk convention regardless of STORAGE_BACKEND: a
	// repeatable, scriptable reset shouldn't depend on live GCS credentials
	// being configured in every environment this might run in.
	LocalStorageDir string
	PublicBaseURL   string
}

// Summary reports what Reset produced — used for the reset endpoint's JSON
// response and the CLI's log output.
type Summary struct {
	Products        int
	Sales           int
	MarketingAssets int
}

// seedRandSeed is fixed, not time-based: two consecutive Reset calls must
// generate the identical set of sales (count, products, quantities,
// timestamps) so a second run is verifiably "exactly the same state, no
// duplicates" rather than merely "some" valid data each time. Every
// previous run's rows are deleted before new ones are inserted (see
// deleteBusinessData), so this determinism is what makes that comparison
// meaningful.
const seedRandSeed = 42

// salesWindow is how far back the generated sales history spreads, so the
// dashboard's revenue-over-time chart has real shape instead of a flat
// line — see CLAUDE.md's seed requirements.
const salesWindow = 14 * 24 * time.Hour

// Reset deletes all business data (never the users table) and reinserts a
// realistic demo catalog, 14 days of sales history, and a marketing
// gallery, all inside a single transaction — either the whole reset lands
// or none of it does. It upserts the demo admin account so that account
// persists across resets instead of being recreated.
func Reset(ctx context.Context, pool *pgxpool.Pool, opts Options) (Summary, error) {
	if opts.AdminEmail == "" || opts.AdminPassword == "" {
		return Summary{}, fmt.Errorf("demoseed: DEMO_ADMIN_EMAIL and DEMO_ADMIN_PASSWORD are required")
	}

	rng := rand.New(rand.NewSource(seedRandSeed))

	imageURLs, err := copySeedAssets(opts.SeedAssetsDir, opts.LocalStorageDir, opts.PublicBaseURL)
	if err != nil {
		return Summary{}, fmt.Errorf("demoseed: copy seed assets: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("demoseed: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := deleteBusinessData(ctx, tx); err != nil {
		return Summary{}, fmt.Errorf("demoseed: delete existing business data: %w", err)
	}

	adminID, err := upsertAdminUser(ctx, tx, opts.AdminEmail, opts.AdminPassword)
	if err != nil {
		return Summary{}, fmt.Errorf("demoseed: upsert admin user: %w", err)
	}

	now := time.Now().UTC()
	windowStart := now.Add(-salesWindow)

	products := demoProducts(imageURLs)
	sales := buildSales(rng, products, windowStart, now)

	productIDs, err := insertProducts(ctx, tx, products, adminID)
	if err != nil {
		return Summary{}, fmt.Errorf("demoseed: insert products: %w", err)
	}

	if err := insertSales(ctx, tx, sales, productIDs, adminID); err != nil {
		return Summary{}, fmt.Errorf("demoseed: insert sales: %w", err)
	}

	if err := insertStockMovements(ctx, tx, products, productIDs, sales, adminID, windowStart); err != nil {
		return Summary{}, fmt.Errorf("demoseed: insert stock movements: %w", err)
	}

	assets := demoMarketingAssets(imageURLs)
	if err := insertMarketingAssets(ctx, tx, assets, adminID); err != nil {
		return Summary{}, fmt.Errorf("demoseed: insert marketing assets: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Summary{}, fmt.Errorf("demoseed: commit: %w", err)
	}

	return Summary{
		Products:        len(products),
		Sales:           len(sales),
		MarketingAssets: len(assets),
	}, nil
}
