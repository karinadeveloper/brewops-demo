import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createTestingPinia } from '@pinia/testing'
import { IDBFactory } from 'fake-indexeddb'
import SalesHistoryView from './SalesHistoryView.vue'

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: postMock, patch: vi.fn(), delete: vi.fn() }),
}))

const isOnlineRef = ref(true)
vi.mock('../composables/useConnectivity', () => ({
  useConnectivity: () => ({ isOnline: isOnlineRef }),
}))

const { addPendingSale } = await import('../composables/useOfflineSalesDb')

async function renderHistory() {
  const pinia = createTestingPinia({ stubActions: false })
  return render(SalesHistoryView, { global: { plugins: [pinia] } })
}

describe('SalesHistoryView', () => {
  beforeEach(() => {
    globalThis.indexedDB = new IDBFactory()
    getMock.mockReset()
    postMock.mockReset()
  })

  it('shows an empty state when there are no sales at all', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])

    // Act
    await renderHistory()

    // Assert
    await waitFor(() => expect(screen.getByText('Todavía no hay ventas')).toBeInTheDocument())
  })

  it('lists a confirmed sale as synced', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([
      { id: 's1', items: [], total_cents: 4500, payment_method: 'CASH', created_at: '2026-01-01T12:00:00Z' },
    ])

    // Act
    await renderHistory()

    // Assert
    await waitFor(() => expect(screen.getByText('Sincronizada')).toBeInTheDocument())
  })

  it('shows a PENDING_SYNC local sale with the correct badge', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])
    await addPendingSale({
      id: 'local-1',
      items: [{ product_id: 'p1', quantity: 1, unit_price_cents: 4500 }],
      payment_method: 'CASH',
      createdAt: '2026-01-02T12:00:00.000Z',
      status: 'PENDING_SYNC',
    })

    // Act
    await renderHistory()

    // Assert
    await waitFor(() => expect(screen.getByText('Pendiente de sincronizar')).toBeInTheDocument())
  })

  it('shows a transport SYNC_ERROR with a working Reintentar button', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])
    await addPendingSale({
      id: 'local-1',
      items: [{ product_id: 'p1', quantity: 1, unit_price_cents: 4500 }],
      payment_method: 'CASH',
      createdAt: '2026-01-02T12:00:00.000Z',
      status: 'SYNC_ERROR',
      errorKind: 'transport',
      errorMessage: 'No se pudo conectar con el servidor.',
    })
    postMock.mockResolvedValueOnce({ id: 'server-1' })
    await renderHistory()
    await waitFor(() => expect(screen.getByText('Error de sincronización')).toBeInTheDocument())
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Reintentar' }))

    // Assert
    await waitFor(() => expect(screen.queryByText('Error de sincronización')).not.toBeInTheDocument())
  })

  it('shows a business SYNC_ERROR without a retry button, only an explanatory note', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])
    await addPendingSale({
      id: 'local-1',
      items: [{ product_id: 'p1', quantity: 1, unit_price_cents: 4500 }],
      payment_method: 'CASH',
      createdAt: '2026-01-02T12:00:00.000Z',
      status: 'SYNC_ERROR',
      errorKind: 'business',
      errorMessage: 'insufficient stock: Jugo de naranja 1L',
    })

    // Act
    await renderHistory()

    // Assert
    await waitFor(() => expect(screen.getByText('insufficient stock: Jugo de naranja 1L')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: 'Reintentar' })).not.toBeInTheDocument()
  })
})
