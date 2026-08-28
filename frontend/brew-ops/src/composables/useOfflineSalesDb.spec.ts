import { beforeEach, describe, expect, it } from 'vitest'
import 'fake-indexeddb/auto'
import { IDBFactory } from 'fake-indexeddb'
import {
  addPendingSale,
  deletePendingSale,
  getAllPendingSales,
  updatePendingSale,
  type PendingSale,
} from './useOfflineSalesDb'

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

describe('useOfflineSalesDb', () => {
  beforeEach(() => {
    // Fresh database per test — fake-indexeddb persists across tests in
    // the same file otherwise, since it simulates real browser storage.
    globalThis.indexedDB = new IDBFactory()
  })

  it('addPendingSale then getAllPendingSales round-trips the record', async () => {
    // Arrange
    const sale = makeSale()

    // Act
    await addPendingSale(sale)
    const all = await getAllPendingSales()

    // Assert
    expect(all).toEqual([sale])
  })

  it('getAllPendingSales returns an empty array when nothing has been stored', async () => {
    // Act
    const all = await getAllPendingSales()

    // Assert
    expect(all).toEqual([])
  })

  it('updatePendingSale overwrites the record in place', async () => {
    // Arrange
    await addPendingSale(makeSale())

    // Act
    await updatePendingSale(makeSale({ status: 'SYNC_ERROR', errorKind: 'transport', errorMessage: 'timeout' }))
    const all = await getAllPendingSales()

    // Assert
    expect(all).toHaveLength(1)
    expect(all[0].status).toBe('SYNC_ERROR')
    expect(all[0].errorMessage).toBe('timeout')
  })

  it('deletePendingSale removes only the targeted record', async () => {
    // Arrange
    await addPendingSale(makeSale({ id: 's1' }))
    await addPendingSale(makeSale({ id: 's2' }))

    // Act
    await deletePendingSale('s1')
    const all = await getAllPendingSales()

    // Assert
    expect(all.map((s) => s.id)).toEqual(['s2'])
  })
})
