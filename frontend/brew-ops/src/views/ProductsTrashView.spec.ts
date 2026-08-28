import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createTestingPinia } from '@pinia/testing'
import ProductsTrashView from './ProductsTrashView.vue'
import type { Product } from '../stores/products'

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: postMock, patch: vi.fn(), delete: vi.fn(), postForm: vi.fn() }),
}))

function makeTrashedProduct(overrides: Partial<Product> = {}): Product {
  return {
    id: 'p1',
    name: 'Jugo de naranja 1L',
    category: 'juice',
    sale_price_cents: 4500,
    cost_cents: 2000,
    current_stock: -5,
    min_stock: 10,
    image_url: null,
    version: 2,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    deleted_at: '2026-01-05T12:00:00Z',
    deleted_by_email: 'owner@brewops.mx',
    ...overrides,
  }
}

async function renderTrashView() {
  const pinia = createTestingPinia({ stubActions: false })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/products/trash', name: 'products-trash', component: ProductsTrashView },
      { path: '/products', name: 'products', component: { template: '<div>products</div>' } },
    ],
  })
  await router.push('/products/trash')
  await router.isReady()
  return render(ProductsTrashView, { global: { plugins: [pinia, router] } })
}

describe('ProductsTrashView', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('lists soft-deleted products with who and when deleted them', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeTrashedProduct()])

    // Act
    await renderTrashView()

    // Assert
    await waitFor(() => expect(screen.getByText('Jugo de naranja 1L')).toBeInTheDocument())
    expect(screen.getByText(/owner@brewops\.mx/)).toBeInTheDocument()
  })

  it('shows an empty state when the trash is empty', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])

    // Act
    await renderTrashView()

    // Assert
    await waitFor(() => expect(screen.getByText('La papelera está vacía')).toBeInTheDocument())
  })

  it('restoring successfully removes the product from the trash list', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeTrashedProduct()])
    postMock.mockResolvedValueOnce(makeTrashedProduct({ deleted_at: null, deleted_by_email: null }))
    await renderTrashView()
    await waitFor(() => expect(screen.getByText('Jugo de naranja 1L')).toBeInTheDocument())
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Restaurar' }))

    // Assert
    await waitFor(() => expect(screen.queryByText('Jugo de naranja 1L')).not.toBeInTheDocument())
    expect(screen.getByText('La papelera está vacía')).toBeInTheDocument()
  })

  it('shows the backend message and a corrective suggestion when restore is rejected for negative stock', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeTrashedProduct()])
    postMock.mockRejectedValueOnce({
      status: 409,
      message: 'cannot restore: current stock is -5, adjust inventory first',
    })
    await renderTrashView()
    await waitFor(() => expect(screen.getByText('Jugo de naranja 1L')).toBeInTheDocument())
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Restaurar' }))

    // Assert
    await waitFor(() => {
      expect(
        screen.getByText('cannot restore: current stock is -5, adjust inventory first'),
      ).toBeInTheDocument()
    })
    expect(screen.getByText('Ajustá el inventario de este producto antes de restaurarlo.')).toBeInTheDocument()
    // The product stays visible in the trash — restore did not silently succeed.
    expect(screen.getByText('Jugo de naranja 1L')).toBeInTheDocument()
  })
})
