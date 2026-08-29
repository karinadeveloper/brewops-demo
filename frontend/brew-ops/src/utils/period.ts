// Period presets for the Dashboard's date-range selector. Boundaries are
// computed from the caller's local clock (in practice America/Mexico_City,
// since BrewOps is single-market) — this is the frontend, so building
// period boundaries from local "today" is exactly the kind of timezone
// handling CLAUDE.md reserves for this layer. The resulting from/to are
// still correct UTC instants once sent to the backend.

export type Period = 'today' | 'last7' | 'month'

export const PERIOD_OPTIONS: { value: Period; label: string }[] = [
  { value: 'today', label: 'Hoy' },
  { value: 'last7', label: 'Últimos 7 días' },
  { value: 'month', label: 'Este mes' },
]

export interface PeriodRange {
  from: string
  to: string
}

// Backend report/sales endpoints treat a missing from/to as "all time" (see
// db/queries/reports.sql), so the selected period must always be sent
// explicitly as RFC3339 timestamps — Date#toISOString() satisfies that.
export function resolvePeriodRange(period: Period, now: Date = new Date()): PeriodRange {
  const to = now
  let from: Date

  switch (period) {
    case 'today':
      from = new Date(now)
      from.setHours(0, 0, 0, 0)
      break
    case 'last7':
      from = new Date(now)
      from.setDate(from.getDate() - 6)
      from.setHours(0, 0, 0, 0)
      break
    case 'month':
      from = new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0)
      break
  }

  return { from: from.toISOString(), to: to.toISOString() }
}

// Formats a bare "YYYY-MM-DD" day string (as returned by GET
// /reports/revenue?group_by=day) into a short label. Deliberately does NOT
// go through a timeZone-aware Intl conversion of an instant: the input is
// already a calendar date, not a UTC timestamp, so reparsing it as UTC and
// re-projecting into America/Mexico_City could shift it onto the wrong day
// (Mexico City is UTC-6). Constructing the Date with the local-time
// constructor (year, monthIndex, day) sidesteps that entirely.
export function formatDayLabel(day: string): string {
  const [year, month, dayOfMonth] = day.split('-').map(Number)
  return new Intl.DateTimeFormat('es-MX', { day: 'numeric', month: 'short' }).format(
    new Date(year, month - 1, dayOfMonth),
  )
}
