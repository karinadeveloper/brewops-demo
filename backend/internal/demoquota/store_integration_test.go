//go:build integration

// Exercises PostgresStore and Checker against a real Postgres database. The
// global 50/day cap is proven by inserting a fixture row directly into
// ai_usage rather than by making 50 real requests — see
// TestChecker_GlobalLimitExceeded_DeniesEvenANewSessionWithNoUsageOfItsOwn.
// Run with:
//
//	go test -tags=integration ./internal/demoquota/...
//
// Requires a reachable Postgres at DATABASE_URL with migrations applied:
//
//	docker-compose up -d postgres
//	migrate -path db/migrations -database "$DATABASE_URL" up
package demoquota_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/demoquota"
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

// cleanupSessionUsage deletes every ai_usage row for sessionID once the
// test finishes — for rows the code under test writes via
// Store.IncrementAndGet/Checker.Check rather than a manual fixture insert
// (which insertFixtureUsageRow already cleans up itself).
func cleanupSessionUsage(t *testing.T, pool *pgxpool.Pool, sessionID uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM ai_usage WHERE session_id = $1`,
			pgtype.UUID{Bytes: sessionID, Valid: true},
		); err != nil {
			t.Errorf("cleanup session ai_usage rows: %v", err)
		}
	})
}

// insertFixtureUsageRow writes a row directly into ai_usage, bypassing
// PostgresStore entirely — a fixture, not a call to the code under test.
// Self-cleaning (t.Cleanup deletes it by its primary key), because the
// date bucket a test picks (see the AddDate offsets below) can otherwise be
// reused across repeated runs against the same persistent test database,
// silently accumulating rows and making a SUM(...) assertion drift upward
// on every rerun instead of staying deterministic.
func insertFixtureUsageRow(t *testing.T, pool *pgxpool.Pool, sessionID uuid.UUID, date time.Time, requestCount int32) {
	t.Helper()
	ctx := context.Background()
	pgDate := pgtype.Date{Time: date, Valid: true}
	pgSessionID := pgtype.UUID{Bytes: sessionID, Valid: true}

	_, err := pool.Exec(ctx,
		`INSERT INTO ai_usage (session_id, usage_date, request_count) VALUES ($1, $2, $3)`,
		pgSessionID, pgDate, requestCount,
	)
	if err != nil {
		t.Fatalf("insert fixture ai_usage row: %v", err)
	}

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM ai_usage WHERE session_id = $1 AND usage_date = $2`,
			pgSessionID, pgDate,
		); err != nil {
			t.Errorf("cleanup fixture ai_usage row: %v", err)
		}
	})
}

func TestPostgresStore_IncrementAndGet_PersistsAndAccumulatesAcrossCalls(t *testing.T) {
	// Arrange
	pool := testPool(t)
	store := demoquota.NewPostgresStore(pool)
	sessionID := uuid.New()
	date := demoquota.UsageDate(time.Now())
	ctx := context.Background()
	cleanupSessionUsage(t, pool, sessionID)

	// Act
	first, err := store.IncrementAndGet(ctx, sessionID, date)
	if err != nil {
		t.Fatalf("first increment: %v", err)
	}
	second, err := store.IncrementAndGet(ctx, sessionID, date)
	if err != nil {
		t.Fatalf("second increment: %v", err)
	}

	// Assert
	if first != 1 {
		t.Errorf("expected the first increment to return 1, got %d", first)
	}
	if second != 2 {
		t.Errorf("expected the second increment to return 2, got %d", second)
	}
}

func TestPostgresStore_SumForDate_SumsAcrossSessions(t *testing.T) {
	// Arrange — a fresh, unique date bucket (today plus a random-ish offset
	// far in the future) so this test's fixture rows can never collide with
	// another test's usage for "today".
	pool := testPool(t)
	date := demoquota.UsageDate(time.Now()).AddDate(1, 0, 0)
	insertFixtureUsageRow(t, pool, uuid.New(), date, 3)
	insertFixtureUsageRow(t, pool, uuid.New(), date, 4)

	// Act
	sum, err := demoquota.NewPostgresStore(pool).SumForDate(context.Background(), date)

	// Assert
	if err != nil {
		t.Fatalf("SumForDate: %v", err)
	}
	if sum != 7 {
		t.Errorf("expected sum 7 across both fixture sessions, got %d", sum)
	}
}

// TestChecker_GlobalLimitExceeded_DeniesEvenANewSessionWithNoUsageOfItsOwn is
// the checklist's "simulate the 50/day global cap without 50 real calls"
// case: one fixture row already puts today's global total over the limit,
// so a session making its very first request of the day — 1, nowhere near
// PerSessionLimit — must still be denied, with the global (not session)
// message.
func TestChecker_GlobalLimitExceeded_DeniesEvenANewSessionWithNoUsageOfItsOwn(t *testing.T) {
	// Arrange
	pool := testPool(t)
	date := demoquota.UsageDate(time.Now()).AddDate(2, 0, 0)
	insertFixtureUsageRow(t, pool, uuid.New(), date, demoquota.GlobalLimit+1)

	// Checker.Check computes usage_date as UsageDate(clock()) itself, so the
	// clock must return a real-looking instant that falls on date's
	// calendar day once localized to America/Mexico_City (UTC-6) — not
	// date's own already-midnight-UTC value, which UsageDate would shift
	// back a day. Midday UTC safely lands on the same day at 06:00 local.
	checker := demoquota.NewCheckerWithClock(demoquota.NewPostgresStore(pool), func() time.Time { return date.Add(12 * time.Hour) })
	newSession := uuid.New()
	cleanupSessionUsage(t, pool, newSession)

	// Act
	result, err := checker.Check(context.Background(), newSession)

	// Assert
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Allowed {
		t.Fatal("expected the global cap to deny a brand-new session")
	}
	if result.Message != demoquota.GlobalLimitMessage {
		t.Errorf("expected the global limit message, got %q", result.Message)
	}
}

func TestChecker_SessionLimitExceeded_DeniesWithSessionMessageBeforeGlobalCapMatters(t *testing.T) {
	// Arrange
	pool := testPool(t)
	date := demoquota.UsageDate(time.Now()).AddDate(3, 0, 0)
	// Checker.Check computes usage_date as UsageDate(clock()) itself, so the
	// clock must return a real-looking instant that falls on date's
	// calendar day once localized to America/Mexico_City (UTC-6) — not
	// date's own already-midnight-UTC value, which UsageDate would shift
	// back a day. Midday UTC safely lands on the same day at 06:00 local.
	checker := demoquota.NewCheckerWithClock(demoquota.NewPostgresStore(pool), func() time.Time { return date.Add(12 * time.Hour) })
	sessionID := uuid.New()
	ctx := context.Background()
	cleanupSessionUsage(t, pool, sessionID)

	// Act — one more than PerSessionLimit, well under GlobalLimit.
	var last demoquota.Result
	for i := int32(0); i < demoquota.PerSessionLimit+1; i++ {
		var err error
		last, err = checker.Check(ctx, sessionID)
		if err != nil {
			t.Fatalf("Check call %d: %v", i, err)
		}
	}

	// Assert
	if last.Allowed {
		t.Fatal("expected the session cap to deny the 6th request")
	}
	if last.Message != demoquota.SessionLimitMessage {
		t.Errorf("expected the session limit message, got %q", last.Message)
	}
}
