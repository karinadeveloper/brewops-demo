package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestResolveTopProductsLimit_VariousInputs_ClampsToExpectedRange(t *testing.T) {
	tests := []struct {
		name      string
		requested int32
		want      int32
	}{
		{"zero uses default", 0, defaultTopProductsLimit},
		{"negative uses default", -5, defaultTopProductsLimit},
		{"within range is kept as-is", 25, 25},
		{"exactly max is kept", maxTopProductsLimit, maxTopProductsLimit},
		{"above max is clamped to max", maxTopProductsLimit + 50, maxTopProductsLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			got := resolveTopProductsLimit(tt.requested)

			// Assert
			if got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestValidateSortBy_VariousInputs_ReturnsExpected(t *testing.T) {
	tests := []struct {
		name               string
		sortBy             string
		wantSortByQuantity bool
		wantErr            error
	}{
		{"empty defaults to revenue", "", false, nil},
		{"explicit revenue", "revenue", false, nil},
		{"quantity", "quantity", true, nil},
		{"unknown value is rejected", "profit", false, domain.ErrInvalidSortBy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			gotSortByQuantity, err := validateSortBy(tt.sortBy)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if err == nil && gotSortByQuantity != tt.wantSortByQuantity {
				t.Fatalf("expected sortByQuantity=%v, got %v", tt.wantSortByQuantity, gotSortByQuantity)
			}
		})
	}
}

func TestReportRevenue_UngroupedRequest_ReturnsSingleTotalFromRepository(t *testing.T) {
	// Arrange
	// Known fixture: two sales of 4500 and 2000 cents would sum to 6500 —
	// the repository (backed by SQL SUM in production) is mocked here to
	// return that known total directly, so the assertion below is checking
	// the service passes it through unmodified, not re-deriving the sum.
	const knownTotal int64 = 6500
	var gotGroupByDay bool
	reports := &mockReportRepository{
		revenueFunc: func(_ context.Context, _, _ *time.Time, groupByDay bool) ([]domain.RevenuePoint, error) {
			gotGroupByDay = groupByDay
			return []domain.RevenuePoint{{TotalCents: knownTotal}}, nil
		},
	}
	svc := NewReportService(reports)

	// Act
	points, err := svc.Revenue(context.Background(), RevenueInput{GroupByDay: false})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotGroupByDay {
		t.Fatal("expected groupByDay=false to reach the repository")
	}
	if len(points) != 1 || points[0].TotalCents != knownTotal {
		t.Fatalf("expected a single point totaling %d, got %+v", knownTotal, points)
	}
}

func TestReportRevenue_GroupedByDayRequest_ReturnsPerDayPointsFromRepository(t *testing.T) {
	// Arrange
	day1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	var gotGroupByDay bool
	reports := &mockReportRepository{
		revenueFunc: func(_ context.Context, _, _ *time.Time, groupByDay bool) ([]domain.RevenuePoint, error) {
			gotGroupByDay = groupByDay
			return []domain.RevenuePoint{
				{Day: &day1, TotalCents: 1000},
				{Day: &day2, TotalCents: 2500},
			}, nil
		},
	}
	svc := NewReportService(reports)

	// Act
	points, err := svc.Revenue(context.Background(), RevenueInput{GroupByDay: true})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !gotGroupByDay {
		t.Fatal("expected groupByDay=true to reach the repository")
	}
	if len(points) != 2 || points[0].TotalCents != 1000 || points[1].TotalCents != 2500 {
		t.Fatalf("unexpected points: %+v", points)
	}
}

func TestReportTopProducts_SortByQuantity_RoutesToQuantityRankingWithClampedLimit(t *testing.T) {
	// Arrange
	productA := uuid.New()
	var gotSortByQuantity bool
	var gotLimit int32
	reports := &mockReportRepository{
		topProductsFunc: func(_ context.Context, _, _ *time.Time, sortByQuantity bool, limit int32) ([]domain.TopProduct, error) {
			gotSortByQuantity, gotLimit = sortByQuantity, limit
			return []domain.TopProduct{{ProductID: productA, Name: "Jugo de mango", QuantitySold: 50, RevenueCents: 12000}}, nil
		},
	}
	svc := NewReportService(reports)

	// Act
	products, err := svc.TopProducts(context.Background(), TopProductsInput{SortBy: "quantity", Limit: 0})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !gotSortByQuantity {
		t.Fatal("expected sort=quantity to route to quantity ranking")
	}
	if gotLimit != defaultTopProductsLimit {
		t.Fatalf("expected default limit %d for Limit=0, got %d", defaultTopProductsLimit, gotLimit)
	}
	if len(products) != 1 || products[0].QuantitySold != 50 {
		t.Fatalf("unexpected products: %+v", products)
	}
}

func TestReportTopProducts_DefaultSort_RoutesToRevenueRanking(t *testing.T) {
	// Arrange
	var gotSortByQuantity bool
	reports := &mockReportRepository{
		topProductsFunc: func(_ context.Context, _, _ *time.Time, sortByQuantity bool, _ int32) ([]domain.TopProduct, error) {
			gotSortByQuantity = sortByQuantity
			return []domain.TopProduct{}, nil
		},
	}
	svc := NewReportService(reports)

	// Act
	_, err := svc.TopProducts(context.Background(), TopProductsInput{})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotSortByQuantity {
		t.Fatal("expected the default sort to rank by revenue, not quantity")
	}
}

func TestReportTopProducts_InvalidSort_RejectsWithoutCallingRepository(t *testing.T) {
	// Arrange
	reports := &mockReportRepository{
		topProductsFunc: func(_ context.Context, _, _ *time.Time, _ bool, _ int32) ([]domain.TopProduct, error) {
			t.Fatal("repository should not be called for an invalid sort value")
			return nil, nil
		},
	}
	svc := NewReportService(reports)

	// Act
	_, err := svc.TopProducts(context.Background(), TopProductsInput{SortBy: "invalid"})

	// Assert
	if !errors.Is(err, domain.ErrInvalidSortBy) {
		t.Fatalf("expected ErrInvalidSortBy, got %v", err)
	}
}

func TestReportInventoryValue_ReturnsRepositoryTotal(t *testing.T) {
	// Arrange
	// Known fixture: e.g. 100 units at 500 cents cost + 50 units at 1000
	// cents cost = 50000 + 50000 = 100000 — computed by hand, the
	// repository (backed by SQL SUM in production) is mocked to return it
	// directly.
	const knownValue int64 = 100000
	reports := &mockReportRepository{
		inventoryValueFunc: func(_ context.Context) (int64, error) {
			return knownValue, nil
		},
	}
	svc := NewReportService(reports)

	// Act
	total, err := svc.InventoryValue(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != knownValue {
		t.Fatalf("expected %d, got %d", knownValue, total)
	}
}
