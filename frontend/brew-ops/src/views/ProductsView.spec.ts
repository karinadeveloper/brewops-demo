import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createRouter, createMemoryHistory } from 'vue-router'
import { createTestingPinia } from '@pinia/testing'
import ProductsView from './ProductsView.vue'
import type { Product } from '../stores/products'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: vi.fn(), patch: vi.fn(), delete: vi.fn(), postForm: vi.fn() }),
}))

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

function buildTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/products', name: 'products', component: ProductsView },
      { path: '/products/trash', name: 'products-trash', component: { template: '<div>trash</div>' } },
    ],
  })
}

// stubActions:false so the real fetchProducts/setCategoryFilter actions run
// (exercising the getters/state this view depends on) while the network
// itself is mocked via useApi above.
async function renderProductsView() {
  const pinia = createTestingPinia({ stubActions: false })
  const router = buildTestRouter()
  await router.push('/products')
  await router.isReady()
  return { ...render(ProductsView, { global: { plugins: [pinia, router] } }), router }
}

describe('ProductsView', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('renders the fetched product list', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct({ name: 'Jugo de naranja 1L' })])

    // Act
    await renderProductsView()

    // Assert
    await waitFor(() => expect(screen.getByText('Jugo de naranja 1L')).toBeInTheDocument())
  })

  it('shows the "no products yet" empty state when the list is genuinely empty', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])

    // Act
    await renderProductsView()

    // Assert
    await waitFor(() => expect(screen.getByText('Aún no tenés productos')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Agregar tu primer producto' })).toBeInTheDocument()
  })

  it('shows the "no results" empty state when a search matches nothing', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct({ name: 'Jugo de naranja 1L' })])
    await renderProductsView()
    await waitFor(() => expect(screen.getByText('Jugo de naranja 1L')).toBeInTheDocument())
    const user = userEvent.setup()

    // Act
    await user.type(screen.getByLabelText('Buscar productos por nombre'), 'no existe')

    // Assert
    await waitFor(() => expect(screen.getByText('Sin resultados')).toBeInTheDocument())
  })

  it('shows the load-error empty state when fetching fails', async () => {
    // Arrange
    getMock.mockRejectedValueOnce({ status: 500, message: 'server error' })

    // Act
    await renderProductsView()

    // Assert
    await waitFor(() => expect(screen.getByText('No se pudieron cargar los productos')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeInTheDocument()
  })

  it('filters by category by re-fetching with the category query param', async () => {
    // Arrange
    getMock.mockResolvedValue([makeProduct()])
    await renderProductsView()
    await waitFor(() => expect(getMock).toHaveBeenCalledTimes(1))
    const user = userEvent.setup()

    // Act
    await user.selectOptions(screen.getByLabelText('Filtrar por categoría'), 'water')

    // Assert
    await waitFor(() => expect(getMock).toHaveBeenCalledTimes(2))
    expect(getMock.mock.calls[1][0]).toContain('category=water')
  })

  it('shows a low-stock badge for a product at or below its minimum stock', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct({ current_stock: 2, min_stock: 10 })])

    // Act
    await renderProductsView()

    // Assert
    await waitFor(() => expect(screen.getByText('Stock bajo')).toBeInTheDocument())
  })

  it('opens the create-product modal from the empty state action', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([])
    await renderProductsView()
    await waitFor(() => expect(screen.getByText('Aún no tenés productos')).toBeInTheDocument())
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Agregar tu primer producto' }))

    // Assert
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Agregar producto' })).toBeInTheDocument()
  })
})
