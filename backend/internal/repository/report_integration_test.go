//go:build integration

// See sale_integration_test.go for why these tests need a real Postgres
// database rather than a mock: this specific case is entirely a SQL
// question (Postgres's own date_trunc/AT TIME ZONE behavior), so a
// mock-based service test could never actually exercise it.
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kariaranelly/brew-ops/backend/internal/repository"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// TestGetRevenueByDay_LateNightSaleInMexicoCity_GroupsIntoTheMexicoCityCalendarDay
// is the fixed-timestamp regression test for the UTC-vs-Mexico_City
// day-bucketing bug: a sale made at 23:30 in Mexico City (America's
// standard UTC-6, no DST since 2022) is already 05:30 the *next* calendar
// day in UTC. Before the fix, GetRevenueByDay's date_trunc('day', ...) ran
// in the DB session's timezone (UTC) and would have bucketed this sale
// into January 16th — one day later than what actually happened from the
// business owner's perspective in Mexico City. This uses a fixed,
// hardcoded timestamp (not time.Now()) so the test is deterministic
// regardless of when it runs.
func TestGetRevenueByDay_LateNightSaleInMexicoCity_GroupsIntoTheMexicoCityCalendarDay(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)

	// 2026-01-15 23:30:00 in Mexico City (UTC-6) == 2026-01-16 05:30:00 UTC.
	lateNightMexicoCity := time.Date(2026, 1, 15, 23, 30, 0, 0, time.FixedZone("Mexico_City", -6*60*60))
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO sales (total_cents, payment_method, created_by, created_at) VALUES ($1, $2, $3, $4)`,
		int64(5000), "CASH", userID, lateNightMexicoCity,
	); err != nil {
		t.Fatalf("failed to insert sale with fixed created_at: %v", err)
	}

	reports := service.NewReportService(repository.NewReportRepository(pool))

	// A tight window around the test date only — this local database
	// accumulates rows from many other integration test runs, so an
	// unbounded query would pick up unrelated sales.
	from := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC)

	// Act
	points, err := reports.Revenue(context.Background(), service.RevenueInput{
		From:       &from,
		To:         &to,
		GroupByDay: true,
	})

	// Assert
	if err != nil {
		t.Fatalf("failed to compute revenue by day: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected exactly 1 day bucket, got %d: %+v", len(points), points)
	}
	if points[0].Day == nil {
		t.Fatal("expected a non-nil day for a grouped revenue point")
	}
	// This is the crux of the fix: the sale must land on January 15th (the
	// Mexico City calendar day it actually happened on), not the 16th (the
	// UTC calendar day the same instant falls on).
	gotDay := points[0].Day.Format("2006-01-02")
	if gotDay != "2026-01-15" {
		t.Fatalf("expected the late-night sale to be grouped into 2026-01-15 (Mexico City calendar day), got %s", gotDay)
	}
	if points[0].TotalCents != 5000 {
		t.Fatalf("expected total_cents 5000, got %d", points[0].TotalCents)
	}
}
