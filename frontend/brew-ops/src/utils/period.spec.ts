import { describe, expect, it } from 'vitest'
import { formatDayLabel, resolvePeriodRange } from './period'

describe('resolvePeriodRange', () => {
  // Fixed "now" so every case is deterministic regardless of when the
  // suite runs: 2026-08-28 15:30:00 local time.
  const now = new Date(2026, 7, 28, 15, 30, 0)

  it('"today" starts at local midnight of the current day', () => {
    const { from, to } = resolvePeriodRange('today', now)

    const fromDate = new Date(from)
    expect(fromDate.getFullYear()).toBe(2026)
    expect(fromDate.getMonth()).toBe(7)
    expect(fromDate.getDate()).toBe(28)
    expect(fromDate.getHours()).toBe(0)
    expect(fromDate.getMinutes()).toBe(0)
    expect(to).toBe(now.toISOString())
  })

  it('"last7" starts at local midnight 6 days before today (7-day inclusive window)', () => {
    const { from, to } = resolvePeriodRange('last7', now)

    const fromDate = new Date(from)
    expect(fromDate.getDate()).toBe(22)
    expect(fromDate.getHours()).toBe(0)
    expect(to).toBe(now.toISOString())
  })

  it('"month" starts at local midnight on the 1st of the current month', () => {
    const { from, to } = resolvePeriodRange('month', now)

    const fromDate = new Date(from)
    expect(fromDate.getFullYear()).toBe(2026)
    expect(fromDate.getMonth()).toBe(7)
    expect(fromDate.getDate()).toBe(1)
    expect(fromDate.getHours()).toBe(0)
    expect(to).toBe(now.toISOString())
  })

  it('"last7" spanning a month boundary rolls into the previous month correctly', () => {
    const earlyMonth = new Date(2026, 8, 3, 10, 0, 0) // Sept 3

    const { from } = resolvePeriodRange('last7', earlyMonth)

    const fromDate = new Date(from)
    expect(fromDate.getMonth()).toBe(7) // August
    expect(fromDate.getDate()).toBe(28)
  })
})

describe('formatDayLabel', () => {
  it('formats a bare YYYY-MM-DD string without shifting the day', () => {
    expect(formatDayLabel('2026-08-01')).toBe('1 ago')
    expect(formatDayLabel('2026-12-31')).toBe('31 dic')
  })
})
