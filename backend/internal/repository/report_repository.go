package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// ReportRepository adapts the sqlc-generated Queries to
// domain.ReportRepository. Every aggregation runs as SQL (SUM/GROUP BY),
// never in Go.
type ReportRepository struct {
	q *Queries
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{q: New(pool)}
}

func (r *ReportRepository) Revenue(ctx context.Context, from, to *time.Time, groupByDay bool) ([]domain.RevenuePoint, error) {
	if !groupByDay {
		total, err := r.q.GetRevenueTotal(ctx, GetRevenueTotalParams{
			FromDate: toTimestamptz(from),
			ToDate:   toTimestamptz(to),
		})
		if err != nil {
			return nil, err
		}
		return []domain.RevenuePoint{{TotalCents: total}}, nil
	}

	rows, err := r.q.GetRevenueByDay(ctx, GetRevenueByDayParams{
		FromDate: toTimestamptz(from),
		ToDate:   toTimestamptz(to),
	})
	if err != nil {
		return nil, err
	}
	points := make([]domain.RevenuePoint, len(rows))
	for i, row := range rows {
		day := row.Day.Time
		points[i] = domain.RevenuePoint{Day: &day, TotalCents: row.TotalCents}
	}
	return points, nil
}

func (r *ReportRepository) TopProducts(ctx context.Context, from, to *time.Time, sortByQuantity bool, limit int32) ([]domain.TopProduct, error) {
	if sortByQuantity {
		rows, err := r.q.ListTopProductsByQuantity(ctx, ListTopProductsByQuantityParams{
			FromDate: toTimestamptz(from),
			ToDate:   toTimestamptz(to),
			Limit:    limit,
		})
		if err != nil {
			return nil, err
		}
		products := make([]domain.TopProduct, len(rows))
		for i, row := range rows {
			products[i] = domain.TopProduct{
				ProductID:    fromUUID(row.ID),
				Name:         row.Name,
				QuantitySold: row.QuantitySold,
				RevenueCents: row.RevenueCents,
			}
		}
		return products, nil
	}

	rows, err := r.q.ListTopProductsByRevenue(ctx, ListTopProductsByRevenueParams{
		FromDate: toTimestamptz(from),
		ToDate:   toTimestamptz(to),
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}
	products := make([]domain.TopProduct, len(rows))
	for i, row := range rows {
		products[i] = domain.TopProduct{
			ProductID:    fromUUID(row.ID),
			Name:         row.Name,
			QuantitySold: row.QuantitySold,
			RevenueCents: row.RevenueCents,
		}
	}
	return products, nil
}

func (r *ReportRepository) InventoryValue(ctx context.Context) (int64, error) {
	return r.q.GetInventoryValue(ctx)
}
