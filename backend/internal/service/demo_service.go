package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/demoseed"
)

// DemoService wraps demoseed.Reset so DemoHandler never touches the
// database directly, consistent with this package's "handlers delegate to
// services" layering rule. Demo-only; only constructed in
// cmd/server/main.go when config.Config.DemoMode is true.
type DemoService struct {
	pool *pgxpool.Pool
	opts demoseed.Options
}

func NewDemoService(pool *pgxpool.Pool, opts demoseed.Options) *DemoService {
	return &DemoService{pool: pool, opts: opts}
}

func (s *DemoService) Reset(ctx context.Context) (demoseed.Summary, error) {
	return demoseed.Reset(ctx, s.pool, s.opts)
}
