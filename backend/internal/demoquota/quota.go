// Package demoquota implements the two-tier daily usage cap on
// POST /inventory/suggest for the public demo deployment of this repo: a
// per-anonymous-session limit and a limit shared across every visitor for
// the day. This exists only to bound real OpenAI spend on a public demo with
// no login wall in front of it — the real BrewOps product has no equivalent
// concept (a single admin's normal usage is the only limit there). See
// CLAUDE.md's "DEMO MODE" section. Nothing in this package is reachable
// unless config.Config.DemoMode is true.
package demoquota

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PerSessionLimit is how many /inventory/suggest calls a single anonymous
// demo session may make in one calendar day (America/Mexico_City).
const PerSessionLimit = 5

// GlobalLimit is how many /inventory/suggest calls all demo sessions
// combined may make in one calendar day (America/Mexico_City).
const GlobalLimit = 50

// SessionLimitMessage is returned when the caller's own session has used up
// its daily quota. The rest of the app remains usable — only this one
// AI-backed endpoint is blocked.
const SessionLimitMessage = "Alcanzaste el límite de pruebas de IA por hoy para esta sesión. El resto de la app sigue disponible."

// GlobalLimitMessage is returned when the shared daily quota across every
// demo visitor has been exhausted, regardless of the caller's own usage.
const GlobalLimitMessage = "Este demo alcanzó su límite diario de pruebas de IA compartido entre todos los visitantes. Volvé a intentarlo mañana, o explorá el resto de la app mientras tanto."

// mexicoCityLocation is loaded once. Falls back to UTC if the local tzdata
// database is unavailable (e.g. a minimal container image) rather than
// panicking — a wrong-by-a-few-hours usage_date bucket is a far smaller
// problem than the quota feature crashing the process.
var mexicoCityLocation = loadMexicoCityLocation()

func loadMexicoCityLocation() *time.Location {
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		return time.UTC
	}
	return loc
}

// UsageDate returns the calendar-day bucket now falls into, in
// America/Mexico_City — consistent with CLAUDE.md's rule that "a day" for
// this app always means a calendar day in that timezone, never in UTC or
// the host's local time.
func UsageDate(now time.Time) time.Time {
	local := now.In(mexicoCityLocation)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// Store is the persistence boundary Checker depends on — implemented by
// PostgresStore against the ai_usage table.
type Store interface {
	// IncrementAndGet atomically increments (session_id, date)'s
	// request_count by one, creating the row if needed, and returns the new
	// count. The increment always happens — Checker.Check relies on this to
	// reserve a slot before the (slow, external) OpenAI call, so the quota
	// is charged exactly once per attempt regardless of whether that call
	// later succeeds or fails.
	IncrementAndGet(ctx context.Context, sessionID uuid.UUID, date time.Time) (int32, error)
	// SumForDate returns the total request_count across every session for
	// date, including any row just written by IncrementAndGet.
	SumForDate(ctx context.Context, date time.Time) (int64, error)
}

// Checker evaluates the two-tier quota for one demo session.
type Checker struct {
	store Store
	now   func() time.Time
}

func NewChecker(store Store) *Checker {
	return &Checker{store: store, now: time.Now}
}

// NewCheckerWithClock is NewChecker with an injectable clock — used by
// integration tests to pin a specific usage_date bucket (so fixture rows
// can't collide with whatever "today" happens to be when the suite runs) or
// to simulate a day boundary. Production code always uses NewChecker.
func NewCheckerWithClock(store Store, now func() time.Time) *Checker {
	return &Checker{store: store, now: now}
}

// Result is the outcome of Check.
type Result struct {
	// Allowed reports whether the caller may proceed to call OpenAI.
	Allowed bool
	// Message is a user-facing (Spanish) explanation, set whenever Allowed
	// is false.
	Message string
}

// Check reserves this request's slot against both quota tiers for
// sessionID's session and today's usage_date. It always increments the
// per-session counter first — see Store.IncrementAndGet — so a rejected
// request still counts against the session (deliberately: it prevents a
// caller from probing the limit for free by retrying after every
// rejection).
func (c *Checker) Check(ctx context.Context, sessionID uuid.UUID) (Result, error) {
	date := UsageDate(c.now())

	sessionCount, err := c.store.IncrementAndGet(ctx, sessionID, date)
	if err != nil {
		return Result{}, fmt.Errorf("demoquota: increment session usage: %w", err)
	}

	globalCount, err := c.store.SumForDate(ctx, date)
	if err != nil {
		return Result{}, fmt.Errorf("demoquota: sum global usage: %w", err)
	}

	return evaluate(sessionCount, globalCount), nil
}

// evaluate is the pure decision logic, kept separate from Check so it's
// testable without a database.
func evaluate(sessionCount int32, globalCount int64) Result {
	if sessionCount > PerSessionLimit {
		return Result{Allowed: false, Message: SessionLimitMessage}
	}
	if globalCount > GlobalLimit {
		return Result{Allowed: false, Message: GlobalLimitMessage}
	}
	return Result{Allowed: true}
}
