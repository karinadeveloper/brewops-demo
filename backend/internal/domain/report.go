package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidSortBy is returned when a top-products sort parameter is
// neither "revenue" nor "quantity".
var ErrInvalidSortBy = errors.New("sort must be \"revenue\" or \"quantity\"")

// RevenuePoint is one row of the revenue report — either the single
// overall total (Day is nil) or one day's total when grouped by day.
type RevenuePoint struct {
	Day        *time.Time
	TotalCents int64
}

// TopProduct is one row of the top-products report.
type TopProduct struct {
	ProductID    uuid.UUID
	Name         string
	QuantitySold int32
	RevenueCents int64
}

// ReportRepository is the persistence boundary the report service depends
// on. Every aggregation (sums, group-by) happens in SQL, never by loading
// rows into Go and summing in memory.
type ReportRepository interface {
	// Revenue returns a single overall total when groupByDay is false, or
	// one RevenuePoint per day with at least one sale when true.
	Revenue(ctx context.Context, from, to *time.Time, groupByDay bool) ([]RevenuePoint, error)
	// TopProducts returns up to limit products, ordered by revenue or by
	// quantity sold depending on sortByQuantity.
	TopProducts(ctx context.Context, from, to *time.Time, sortByQuantity bool, limit int32) ([]TopProduct, error)
	// InventoryValue returns current_stock * cost_cents summed across every
	// active (non-soft-deleted) product.
	InventoryValue(ctx context.Context) (int64, error)
}
