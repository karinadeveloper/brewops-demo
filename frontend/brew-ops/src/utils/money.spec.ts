import { describe, expect, it } from 'vitest'
import { centsToPesos, formatCentsAsPesos, pesosToCents } from './money'

describe('pesosToCents', () => {
  it.each([
    [45, 4500],
    [0, 0],
    [12.5, 1250],
    [12.34, 1234],
    [10.005, 1001], // rounds rather than truncates
  ])('converts %d pesos to %d cents', (pesos, expected) => {
    expect(pesosToCents(pesos)).toBe(expected)
  })
})

describe('centsToPesos', () => {
  it.each([
    [4500, 45],
    [0, 0],
    [1250, 12.5],
  ])('converts %d cents to %d pesos', (cents, expected) => {
    expect(centsToPesos(cents)).toBe(expected)
  })
})

describe('formatCentsAsPesos', () => {
  it('formats cents as an MXN currency string', () => {
    expect(formatCentsAsPesos(4500)).toBe('$45.00')
  })

  it('formats zero correctly', () => {
    expect(formatCentsAsPesos(0)).toBe('$0.00')
  })
})
