import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: vi.fn(), patch: vi.fn(), delete: vi.fn() }),
}))

const { useDashboardStore } = await import('./dashboard')

function mockGetByPath(responses: Record<string, unknown>) {
  getMock.mockImplementation((path: string) => {
    const entry = Object.entries(responses).find(([prefix]) => path.startsWith(prefix))
    if (!entry) {
      return Promise.reject(new Error(`unexpected path: ${path}`))
    }
    return Promise.resolve(entry[1])
  })
}

describe('useDashboardStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getMock.mockReset()
  })

  it('loadRevenue stores the points and sums total_cents client-side', async () => {
    mockGetByPath({
      '/reports/revenue': [
        { day: '2026-08-27', total_cents: 1000 },
        { day: '2026-08-28', total_cents: 2500 },
      ],
    })
    const store = useDashboardStore()

    await store.loadRevenue()

    expect(store.revenuePoints).toHaveLength(2)
    expect(store.revenueTotalCents).toBe(3500)
    expect(store.revenueError).toBe(false)
    expect(getMock).toHaveBeenCalledWith(expect.stringContaining('group_by=day'))
  })

  it('loadRevenue sets revenueError on failure without touching other sections state', async () => {
    getMock.mockRejectedValueOnce(new Error('boom'))
    const store = useDashboardStore()

    await store.loadRevenue()

    expect(store.revenueError).toBe(true)
    expect(store.isLoadingRevenue).toBe(false)
  })

  it('loadSalesCount sums across pages until a short page ends pagination', async () => {
    let call = 0
    getMock.mockImplementation((path: string) => {
      expect(path).toContain('/sales?')
      call += 1
      if (call === 1) {
        return Promise.resolve(Array.from({ length: 100 }, () => ({})))
      }
      return Promise.resolve(Array.from({ length: 37 }, () => ({})))
    })
    const store = useDashboardStore()

    await store.loadSalesCount()

    expect(store.salesCount).toBe(137)
    expect(call).toBe(2)
  })

  it('loadSalesCount stops immediately when the first page is already short', async () => {
    getMock.mockResolvedValueOnce(Array.from({ length: 5 }, () => ({})))
    const store = useDashboardStore()

    await store.loadSalesCount()

    expect(store.salesCount).toBe(5)
    expect(getMock).toHaveBeenCalledTimes(1)
  })

  it('loadInventoryValue and loadLowStock are independent of the selected period', async () => {
    mockGetByPath({
      '/reports/inventory-value': { total_value_cents: 50000 },
      '/products/low-stock': [{ id: 'p1', current_stock: 1, min_stock: 5 }],
    })
    const store = useDashboardStore()

    await Promise.all([store.loadInventoryValue(), store.loadLowStock()])

    expect(store.inventoryValueCents).toBe(50000)
    expect(store.lowStockItems).toHaveLength(1)
    // Neither call includes from/to — these are current-moment snapshots.
    for (const call of getMock.mock.calls) {
      expect(call[0]).not.toContain('from=')
    }
  })

  it('loadTopProducts stores the returned ranking', async () => {
    mockGetByPath({
      '/reports/top-products': [
        { product_id: 'p1', name: 'Jugo de mango', quantity_sold: 10, revenue_cents: 15000 },
      ],
    })
    const store = useDashboardStore()

    await store.loadTopProducts()

    expect(store.topProducts).toHaveLength(1)
    expect(store.topProductsError).toBe(false)
  })

  it('setPeriod updates the period and reloads only the period-dependent sections', async () => {
    mockGetByPath({
      '/reports/revenue': [],
      '/sales': [],
      '/reports/top-products': [],
    })
    const store = useDashboardStore()
    getMock.mockClear()

    store.setPeriod('month')
    await vi.waitFor(() => expect(store.isLoadingRevenue).toBe(false))

    expect(store.period).toBe('month')
    const calledPaths = getMock.mock.calls.map(([path]) => path as string)
    expect(calledPaths.some((p) => p.includes('/reports/revenue'))).toBe(true)
    expect(calledPaths.some((p) => p.includes('/sales'))).toBe(true)
    expect(calledPaths.some((p) => p.includes('/reports/top-products'))).toBe(true)
    expect(calledPaths.some((p) => p.includes('/reports/inventory-value'))).toBe(false)
    expect(calledPaths.some((p) => p.includes('/products/low-stock'))).toBe(false)
  })

  it('loadAll loads every section and one failing section does not affect the others', async () => {
    getMock.mockImplementation((path: string) => {
      if (path.startsWith('/reports/top-products')) {
        return Promise.reject(new Error('boom'))
      }
      if (path.startsWith('/reports/revenue')) {
        return Promise.resolve([{ day: '2026-08-28', total_cents: 100 }])
      }
      if (path.startsWith('/sales')) {
        return Promise.resolve([])
      }
      if (path.startsWith('/reports/inventory-value')) {
        return Promise.resolve({ total_value_cents: 999 })
      }
      if (path.startsWith('/products/low-stock')) {
        return Promise.resolve([])
      }
      return Promise.reject(new Error(`unexpected path: ${path}`))
    })
    const store = useDashboardStore()

    await store.loadAll()

    expect(store.topProductsError).toBe(true)
    expect(store.revenueError).toBe(false)
    expect(store.revenueTotalCents).toBe(100)
    expect(store.inventoryValueCents).toBe(999)
  })
})
