import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { IDBFactory } from 'fake-indexeddb'
import type { PendingSale } from './useOfflineSalesDb'

// postMock must come from vi.hoisted (referenced both inside the mock
// factory below and in test bodies for assertions). isOnlineRef must NOT —
// vi.hoisted's callback runs immediately at the hoisted position, before
// the `ref` import from 'vue' is bound, so ref(false) would hit a TDZ
// error there. A plain top-level const works because vi.mock's factory
// itself only runs lazily, on first import of the mocked module — by then
// isOnlineRef is already assigned.
const { postMock } = vi.hoisted(() => ({ postMock: vi.fn() }))

const isOnlineRef = ref(false)

vi.mock('./useConnectivity', () => ({
  useConnectivity: () => ({ isOnline: isOnlineRef }),
}))

vi.mock('./useApi', () => ({
  useApi: () => ({ get: vi.fn(), post: postMock, patch: vi.fn(), delete: vi.fn() }),
}))

const { syncPendingSales, retryPendingSale, useSaleSync, __resetSaleSyncForTests } = await import('./useSaleSync')
const { addPendingSale, getAllPendingSales } = await import('./useOfflineSalesDb')

function makeSale(overrides: Partial<PendingSale> = {}): PendingSale {
  return {
    id: 's1',
    items: [{ product_id: 'p1', quantity: 2, unit_price_cents: 4500 }],
    payment_method: 'CASH',
    idempotencyKey: 'key-1',
    createdAt: '2026-01-01T00:00:00.000Z',
    status: 'PENDING_SYNC',
    ...overrides,
  }
}

describe('useSaleSync', () => {
  beforeEach(() => {
    globalThis.indexedDB = new IDBFactory()
    postMock.mockReset()
    isOnlineRef.value = false
    __resetSaleSyncForTests()
  })

  it('a successful sync removes the sale from local storage', async () => {
    // Arrange
    await addPendingSale(makeSale())
    postMock.mockResolvedValueOnce({ id: 'server-1' })

    // Act
    await syncPendingSales()

    // Assert
    expect(await getAllPendingSales()).toEqual([])
    expect(postMock).toHaveBeenCalledWith('/sales', {
      items: [{ product_id: 'p1', quantity: 2, unit_price_cents: 4500 }],
      payment_method: 'CASH',
      idempotency_key: 'key-1',
    })
  })

  it('a business rejection marks SYNC_ERROR with kind "business", alerts the caller, and is never auto-retried', async () => {
    // Arrange
    await addPendingSale(makeSale())
    postMock.mockRejectedValueOnce({ status: 409, message: 'insufficient stock: Jugo de mango' })
    const onRejection = vi.fn()

    // Act
    await syncPendingSales(onRejection)

    // Assert
    const all = await getAllPendingSales()
    expect(all).toHaveLength(1)
    expect(all[0].status).toBe('SYNC_ERROR')
    expect(all[0].errorKind).toBe('business')
    expect(all[0].errorMessage).toBe('insufficient stock: Jugo de mango')
    expect(onRejection).toHaveBeenCalledWith(
      expect.objectContaining({ id: 's1' }),
      'insufficient stock: Jugo de mango',
    )

    // Act — a second sync pass must not touch it again (it's no longer
    // PENDING_SYNC, and business rejections are never blindly retried).
    postMock.mockClear()
    await syncPendingSales()

    // Assert
    expect(postMock).not.toHaveBeenCalled()
  })

  it('a transport failure marks SYNC_ERROR with kind "transport", and retryPendingSale can recover it', async () => {
    // Arrange
    await addPendingSale(makeSale())
    postMock.mockRejectedValueOnce(new TypeError('Failed to fetch'))

    // Act
    await syncPendingSales()

    // Assert
    let all = await getAllPendingSales()
    expect(all[0].status).toBe('SYNC_ERROR')
    expect(all[0].errorKind).toBe('transport')

    // Act — retrying (e.g. the user clicking "Reintentar") succeeds this time
    postMock.mockResolvedValueOnce({ id: 'server-1' })
    await retryPendingSale('s1')

    // Assert
    all = await getAllPendingSales()
    expect(all).toEqual([])
  })

  it('sends the same idempotency_key on every retry of the same local sale, never a fresh one', async () => {
    // Arrange — this is the guarantee that makes queuing a sale offline
    // after a lost transport response safe rather than a duplicate risk.
    await addPendingSale(makeSale({ idempotencyKey: 'stable-key' }))
    postMock.mockRejectedValueOnce(new TypeError('Failed to fetch'))

    // Act — first attempt fails at the transport level.
    await syncPendingSales()
    postMock.mockResolvedValueOnce({ id: 'server-1' })
    await retryPendingSale('s1')

    // Assert — both attempts carried the exact same idempotency_key.
    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/sales',
      expect.objectContaining({ idempotency_key: 'stable-key' }),
    )
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/sales',
      expect.objectContaining({ idempotency_key: 'stable-key' }),
    )
  })

  it('two distinct local sales carry two distinct idempotency keys', async () => {
    // Arrange
    await addPendingSale(makeSale({ id: 's1', idempotencyKey: 'key-a', createdAt: '2026-01-01T00:00:01.000Z' }))
    await addPendingSale(makeSale({ id: 's2', idempotencyKey: 'key-b', createdAt: '2026-01-01T00:00:02.000Z' }))
    postMock.mockResolvedValue({ id: 'ok' })

    // Act
    await syncPendingSales()

    // Assert
    const sentKeys = postMock.mock.calls.map(([, body]) => (body as { idempotency_key: string }).idempotency_key)
    expect(sentKeys).toEqual(['key-a', 'key-b'])
  })

  it('syncs pending sales one at a time, in creation order, never in parallel', async () => {
    // Arrange
    await addPendingSale(makeSale({ id: 's2', createdAt: '2026-01-01T00:00:02.000Z', items: [{ product_id: 'p2', quantity: 1, unit_price_cents: 1000 }] }))
    await addPendingSale(makeSale({ id: 's1', createdAt: '2026-01-01T00:00:01.000Z', items: [{ product_id: 'p1', quantity: 1, unit_price_cents: 1000 }] }))

    let resolveFirst!: (value: unknown) => void
    const dispatchedProducts: string[] = []
    postMock.mockImplementationOnce((_path: string, body: { items: { product_id: string }[] }) => {
      dispatchedProducts.push(body.items[0].product_id)
      return new Promise((resolve) => {
        resolveFirst = resolve
      })
    })
    postMock.mockImplementationOnce((_path: string, body: { items: { product_id: string }[] }) => {
      dispatchedProducts.push(body.items[0].product_id)
      return Promise.resolve({ id: 'ok' })
    })

    // Act
    const syncPromise = syncPendingSales()
    await vi.waitFor(() => expect(postMock).toHaveBeenCalledTimes(1))

    // Assert — the second sale (s2) must not have been dispatched yet while
    // the first (s1, the older one) is still in flight.
    expect(postMock).toHaveBeenCalledTimes(1)
    expect(dispatchedProducts).toEqual(['p1'])

    resolveFirst({ id: 'ok' })
    await syncPromise

    expect(dispatchedProducts).toEqual(['p1', 'p2'])
  })

  it('automatically syncs when connectivity transitions from offline to online', async () => {
    // Arrange
    await addPendingSale(makeSale())
    postMock.mockResolvedValueOnce({ id: 'server-1' })
    isOnlineRef.value = false

    // Act
    useSaleSync()
    isOnlineRef.value = true

    // Assert
    await vi.waitFor(async () => expect(await getAllPendingSales()).toEqual([]))
  })

  // Found via Session 9's offline E2E test: a sale that finished syncing
  // automatically on reconnect (not via this view's own retry button) kept
  // showing as pending in SalesHistoryView, because nothing told it to
  // re-read IndexedDB. syncVersion is the fix — a view that watches it
  // learns a sync pass (background or manual) just finished.
  it('syncVersion increments after a sync pass completes, so other views can react to a background sync', async () => {
    // Arrange
    await addPendingSale(makeSale())
    postMock.mockResolvedValueOnce({ id: 'server-1' })
    const { syncVersion } = useSaleSync()
    const before = syncVersion.value

    // Act
    await syncPendingSales()

    // Assert
    expect(syncVersion.value).toBe(before + 1)
  })

  it('syncVersion also increments after a manual retryPendingSale', async () => {
    // Arrange
    await addPendingSale(makeSale())
    postMock.mockResolvedValueOnce({ id: 'server-1' })
    const { syncVersion } = useSaleSync()
    const before = syncVersion.value

    // Act
    await retryPendingSale('s1')

    // Assert
    expect(syncVersion.value).toBe(before + 1)
  })

  it('does not sync again just because isOnline stays true', async () => {
    // Arrange
    isOnlineRef.value = true
    useSaleSync()
    await addPendingSale(makeSale())
    postMock.mockResolvedValueOnce({ id: 'server-1' })

    // Act — re-triggering the watcher with the same "true" value (no
    // false->true transition) must not start a sync on its own.
    isOnlineRef.value = true

    // Assert
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(postMock).not.toHaveBeenCalled()
  })
})
