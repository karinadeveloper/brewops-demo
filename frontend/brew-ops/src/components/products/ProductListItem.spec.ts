import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import ProductListItem from './ProductListItem.vue'
import type { Product } from '../../stores/products'

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

describe('ProductListItem', () => {
  it('renders the product name, category, and price', () => {
    // Arrange & Act
    render(ProductListItem, { props: { product: makeProduct() } })

    // Assert
    expect(screen.getByText('Jugo de naranja 1L')).toBeInTheDocument()
    expect(screen.getByText('Jugo · $45.00')).toBeInTheDocument()
  })

  it('shows a low-stock badge when current_stock <= min_stock', () => {
    // Arrange & Act
    render(ProductListItem, { props: { product: makeProduct({ current_stock: 5, min_stock: 10 }) } })

    // Assert
    expect(screen.getByText('Stock bajo')).toBeInTheDocument()
  })

  it('does not show a low-stock badge when stock is healthy', () => {
    // Arrange & Act
    render(ProductListItem, { props: { product: makeProduct({ current_stock: 50, min_stock: 10 }) } })

    // Assert
    expect(screen.queryByText('Stock bajo')).not.toBeInTheDocument()
  })

  it('emits edit with the product when the edit button is clicked', async () => {
    // Arrange
    const product = makeProduct()
    const { emitted } = render(ProductListItem, { props: { product } })
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Editar' }))

    // Assert
    expect(emitted().edit).toBeTruthy()
    expect(emitted().edit[0]).toEqual([product])
  })

  it('emits delete with the product when the delete button is clicked', async () => {
    // Arrange
    const product = makeProduct()
    const { emitted } = render(ProductListItem, { props: { product } })
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Eliminar' }))

    // Assert
    expect(emitted().delete).toBeTruthy()
    expect(emitted().delete[0]).toEqual([product])
  })
})
