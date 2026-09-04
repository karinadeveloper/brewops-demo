package service

import (
	"context"
	"time"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

const (
	defaultTopProductsLimit int32 = 10
	maxTopProductsLimit     int32 = 100
)

// ReportService implements the read-only aggregation endpoints
// (revenue, top-products, inventory-value) on top of a
// domain.ReportRepository. Every sum/group-by happens in SQL; this layer
// only validates input and picks which query to run.
type ReportService struct {
	reports domain.ReportRepository
}

func NewReportService(reports domain.ReportRepository) *ReportService {
	return &ReportService{reports: reports}
}

// resolveTopProductsLimit clamps the requested limit into [1, 100],
// defaulting to 10 when none is requested. Pure and exhaustively tested.
func resolveTopProductsLimit(requested int32) int32 {
	if requested <= 0 {
		return defaultTopProductsLimit
	}
	if requested > maxTopProductsLimit {
		return maxTopProductsLimit
	}
	return requested
}

// validateSortBy resolves the ?sort= query param into the boolean the
// repository needs, defaulting to revenue ranking. Pure and exhaustively
// tested — this is the only branching logic behind "sorts by revenue by
// default, ?sort=quantity switches to unit-count ranking."
func validateSortBy(sortBy string) (sortByQuantity bool, err error) {
	switch sortBy {
	case "", "revenue":
		return false, nil
	case "quantity":
		return true, nil
	default:
		return false, domain.ErrInvalidSortBy
	}
}

type RevenueInput struct {
	From       *time.Time
	To         *time.Time
	GroupByDay bool
}

func (s *ReportService) Revenue(ctx context.Context, in RevenueInput) ([]domain.RevenuePoint, error) {
	return s.reports.Revenue(ctx, in.From, in.To, in.GroupByDay)
}

type TopProductsInput struct {
	From   *time.Time
	To     *time.Time
	SortBy string
	Limit  int32
}

func (s *ReportService) TopProducts(ctx context.Context, in TopProductsInput) ([]domain.TopProduct, error) {
	sortByQuantity, err := validateSortBy(in.SortBy)
	if err != nil {
		return nil, err
	}
	return s.reports.TopProducts(ctx, in.From, in.To, sortByQuantity, resolveTopProductsLimit(in.Limit))
}

func (s *ReportService) InventoryValue(ctx context.Context) (int64, error) {
	return s.reports.InventoryValue(ctx)
}
