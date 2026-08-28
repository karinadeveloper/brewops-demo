import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createTestingPinia } from '@pinia/testing'
import { IDBFactory } from 'fake-indexeddb'
import PointOfSaleView from './PointOfSaleView.vue'
import type { Product } from '../stores/products'

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: postMock, patch: vi.fn(), delete: vi.fn() }),
}))

// Plain ref (not vi.hoisted) — see useSaleSync.spec.ts for why: vi.hoisted
// callbacks run before the `ref` import from 'vue' is bound.
const isOnlineRef = ref(true)
vi.mock('../composables/useConnectivity', () => ({
  useConnectivity: () => ({ isOnline: isOnlineRef }),
}))

const { getAllPendingSales } = await import('../composables/useOfflineSalesDb')

function makeProduct(overrides: Partial<Product> = {}): Product {
  return {
    id: 'p1',
    name: 'Jugo de naranja 1L',
    category: 'juice',
    sale_price_cents: 4500,
    cost_cents: 2000,
    current_stock: 50,
    min_stock: 10,
    image_url: null,
    version: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

async function renderPos() {
  const pinia = createTestingPinia({ stubActions: false })
  return render(PointOfSaleView, { global: { plugins: [pinia] } })
}

describe('PointOfSaleView', () => {
  beforeEach(() => {
    globalThis.indexedDB = new IDBFactory()
    getMock.mockReset()
    postMock.mockReset()
    isOnlineRef.value = true
  })

  it('shows an empty state when there are no active products', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])

    // Act
    await renderPos()

    // Assert
    await waitFor(() => expect(screen.getByText('No hay productos disponibles')).toBeInTheDocument())
  })

  it('tapping a product adds it to the cart', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))

    // Assert
    expect(screen.getByLabelText('Cantidad de Jugo de naranja 1L')).toHaveValue(1)
  })

  it('tapping the same product again increments its quantity instead of duplicating it', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()
    const productButton = screen.getByRole('button', { name: /Jugo de naranja/ })

    // Act
    await user.click(productButton)
    await user.click(productButton)

    // Assert
    expect(screen.getByLabelText('Cantidad de Jugo de naranja 1L')).toHaveValue(2)
  })

  it('removing a product from the cart clears its quantity input', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))

    // Act
    await user.click(screen.getByRole('button', { name: 'Quitar Jugo de naranja 1L' }))

    // Assert
    expect(screen.queryByLabelText('Cantidad de Jugo de naranja 1L')).not.toBeInTheDocument()
  })

  it('disables the confirm button for an empty cart and enables it once something is added', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())

    // Assert (before)
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()

    // Act
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))

    // Assert (after)
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).not.toBeDisabled()
  })

  it('lets the user pick the payment method before confirming', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByLabelText('Transferencia'))

    // Assert
    expect(screen.getByLabelText('Transferencia')).toBeChecked()
    expect(screen.getByLabelText('Efectivo')).not.toBeChecked()
  })

  it('confirming an online sale posts to /sales and clears the cart', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    postMock.mockResolvedValueOnce({ id: 's1', items: [], total_cents: 4500, payment_method: 'CASH', created_at: '2026-01-01T00:00:00Z' })
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))

    // Act
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    // Assert
    await waitFor(() => expect(screen.getByText('Venta registrada.')).toBeInTheDocument())
    expect(screen.queryByLabelText('Cantidad de Jugo de naranja 1L')).not.toBeInTheDocument()
    expect(postMock).toHaveBeenCalledWith('/sales', {
      items: [{ product_id: 'p1', quantity: 1, unit_price_cents: 4500 }],
      payment_method: 'CASH',
      idempotency_key: expect.any(String),
    })
  })

  it('a business rejection (409) shows the backend message and keeps the cart intact — not treated as offline', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    postMock.mockRejectedValueOnce({ status: 409, message: 'insufficient stock: Jugo de naranja 1L' })
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))

    // Act
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    // Assert
    await waitFor(() => expect(screen.getByText('insufficient stock: Jugo de naranja 1L')).toBeInTheDocument())
    expect(screen.getByLabelText('Cantidad de Jugo de naranja 1L')).toBeInTheDocument()
    expect(await getAllPendingSales()).toEqual([])
  })

  it('confirming while offline saves the sale to IndexedDB as PENDING_SYNC instead of calling the API', async () => {
    // Arrange
    isOnlineRef.value = false
    getMock.mockResolvedValueOnce([makeProduct()])
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))

    // Act
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    // Assert
    await waitFor(() => expect(screen.getByText(/Se sincronizará cuando vuelva la conexión/)).toBeInTheDocument())
    expect(postMock).not.toHaveBeenCalled()
    const pending = await getAllPendingSales()
    expect(pending).toHaveLength(1)
    expect(pending[0].status).toBe('PENDING_SYNC')
    expect(pending[0].items).toEqual([{ product_id: 'p1', quantity: 1, unit_price_cents: 4500 }])
    expect(pending[0].idempotencyKey).toEqual(expect.any(String))
    expect(screen.queryByLabelText('Cantidad de Jugo de naranja 1L')).not.toBeInTheDocument()
  })

  it('two separate offline confirmations generate two distinct idempotency keys', async () => {
    // Arrange
    isOnlineRef.value = false
    getMock.mockResolvedValueOnce([makeProduct()])
    await renderPos()
    await waitFor(() => expect(screen.getByRole('button', { name: /Jugo de naranja/ })).toBeInTheDocument())
    const user = userEvent.setup()

    // Act — confirm the same product as two separate sales, one at a time.
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await waitFor(async () => expect(await getAllPendingSales()).toHaveLength(1))
    await user.click(screen.getByRole('button', { name: /Jugo de naranja/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await waitFor(async () => expect(await getAllPendingSales()).toHaveLength(2))

    // Assert
    const pending = await getAllPendingSales()
    expect(pending[0].idempotencyKey).not.toBe(pending[1].idempotencyKey)
  })
})
