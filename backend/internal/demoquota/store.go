package demoquota

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store against the ai_usage table using raw SQL —
// deliberately not sqlc-generated like the rest of internal/repository, so
// every demo-only query stays physically isolated in this package instead
// of mixed into the schema the real product's repository layer would reuse.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) IncrementAndGet(ctx context.Context, sessionID uuid.UUID, date time.Time) (int32, error) {
	const query = `
		INSERT INTO ai_usage (session_id, usage_date, request_count)
		VALUES ($1, $2, 1)
		ON CONFLICT (session_id, usage_date)
		DO UPDATE SET request_count = ai_usage.request_count + 1
		RETURNING request_count`

	var count int32
	if err := s.pool.QueryRow(ctx, query, pgtype.UUID{Bytes: sessionID, Valid: true}, pgDate(date)).Scan(&count); err != nil {
		return 0, fmt.Errorf("demoquota: upsert usage: %w", err)
	}
	return count, nil
}

func (s *PostgresStore) SumForDate(ctx context.Context, date time.Time) (int64, error) {
	const query = `SELECT COALESCE(SUM(request_count), 0) FROM ai_usage WHERE usage_date = $1`

	var sum int64
	if err := s.pool.QueryRow(ctx, query, pgDate(date)).Scan(&sum); err != nil {
		return 0, fmt.Errorf("demoquota: sum usage: %w", err)
	}
	return sum, nil
}

// pgDate wraps date for a DATE column parameter. pgx v5 only registers a
// direct time.Time codec for timestamp/timestamptz — a bare time.Time
// passed against a DATE placeholder fails to find an encode plan, since
// pgtype.DateCodec.PlanEncode requires a DateValuer, which time.Time does
// not implement.
func pgDate(date time.Time) pgtype.Date {
	return pgtype.Date{Time: date, Valid: true}
}
