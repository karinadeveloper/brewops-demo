import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: postMock, patch: vi.fn(), delete: vi.fn() }),
}))

const { useSalesStore } = await import('./sales')

describe('useSalesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getMock.mockReset()
    postMock.mockReset()
  })

  it('fetchSales loads items and tracks hasMorePages from a full page', async () => {
    const fullPage = Array.from({ length: 20 }, (_, i) => ({
      id: `s${i}`,
      items: [],
      total_cents: 1000,
      payment_method: 'CASH' as const,
      created_at: '2026-01-01T00:00:00Z',
    }))
    getMock.mockResolvedValueOnce(fullPage)
    const store = useSalesStore()

    await store.fetchSales(1)

    expect(store.items).toHaveLength(20)
    expect(store.hasMorePages).toBe(true)
    expect(store.loadError).toBe(false)
  })

  it('fetchSales sets loadError on failure', async () => {
    getMock.mockRejectedValueOnce({ status: 500, message: 'server error' })
    const store = useSalesStore()

    await store.fetchSales(1)

    expect(store.loadError).toBe(true)
    expect(store.hasLoadedOnce).toBe(true)
  })

  it('setDateRange sends from/to as query params', async () => {
    getMock.mockResolvedValue([])
    const store = useSalesStore()

    store.setDateRange('2026-01-01', '2026-01-31')
    await vi.waitFor(() => expect(getMock).toHaveBeenCalled())

    expect(getMock).toHaveBeenCalledWith(expect.stringContaining('from=2026-01-01'))
    expect(getMock).toHaveBeenCalledWith(expect.stringContaining('to=2026-01-31'))
  })

  it('createSale posts to /sales and prepends the result', async () => {
    const created = {
      id: 's1',
      items: [{ product_id: 'p1', quantity: 2, unit_price_cents: 4500 }],
      total_cents: 9000,
      payment_method: 'CASH' as const,
      created_at: '2026-01-01T00:00:00Z',
    }
    postMock.mockResolvedValueOnce(created)
    const store = useSalesStore()

    const result = await store.createSale({
      items: [{ product_id: 'p1', quantity: 2, unit_price_cents: 4500 }],
      payment_method: 'CASH',
    })

    expect(postMock).toHaveBeenCalledWith('/sales', {
      items: [{ product_id: 'p1', quantity: 2, unit_price_cents: 4500 }],
      payment_method: 'CASH',
    })
    expect(result).toEqual(created)
    expect(store.items[0]).toEqual(created)
  })

  it('createSale propagates a 409 (insufficient stock) without mutating items', async () => {
    postMock.mockRejectedValueOnce({ status: 409, message: 'insufficient stock: Jugo de mango' })
    const store = useSalesStore()

    await expect(
      store.createSale({ items: [{ product_id: 'p1', quantity: 999, unit_price_cents: 4500 }], payment_method: 'CASH' }),
    ).rejects.toMatchObject({ status: 409 })
    expect(store.items).toEqual([])
  })

  it('fetchSaleDetail requests the individual sale', async () => {
    const detail = {
      id: 's1',
      items: [{ product_id: 'p1', quantity: 2, unit_price_cents: 4500 }],
      total_cents: 9000,
      payment_method: 'CASH' as const,
      created_at: '2026-01-01T00:00:00Z',
    }
    getMock.mockResolvedValueOnce(detail)
    const store = useSalesStore()

    const result = await store.fetchSaleDetail('s1')

    expect(getMock).toHaveBeenCalledWith('/sales/s1')
    expect(result).toEqual(detail)
  })
})
