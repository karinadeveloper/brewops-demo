import { describe, expect, it } from 'vitest'
import { useCart } from './useCart'
import type { Product } from '../stores/products'

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

describe('useCart', () => {
  it('starts empty', () => {
    const cart = useCart()
    expect(cart.isEmpty.value).toBe(true)
    expect(cart.totalCents.value).toBe(0)
    expect(cart.itemCount.value).toBe(0)
  })

  it('adding a product for the first time creates a quantity-1 line', () => {
    const cart = useCart()
    cart.addProduct(makeProduct())
    expect(cart.items.value).toEqual([
      { productId: 'p1', name: 'Jugo de naranja 1L', unitPriceCents: 4500, quantity: 1, imageUrl: null },
    ])
  })

  it('adding the same product again increments its quantity instead of duplicating the line', () => {
    const cart = useCart()
    const product = makeProduct()
    cart.addProduct(product)
    cart.addProduct(product)
    expect(cart.items.value).toHaveLength(1)
    expect(cart.items.value[0].quantity).toBe(2)
  })

  it('computes the total across multiple distinct products', () => {
    const cart = useCart()
    cart.addProduct(makeProduct({ id: 'p1', sale_price_cents: 4500 }))
    cart.addProduct(makeProduct({ id: 'p2', sale_price_cents: 2000 }))
    cart.setQuantity('p1', 3)
    expect(cart.totalCents.value).toBe(3 * 4500 + 2000)
    expect(cart.itemCount.value).toBe(4)
  })

  it('removeItem drops the line entirely', () => {
    const cart = useCart()
    cart.addProduct(makeProduct({ id: 'p1' }))
    cart.removeItem('p1')
    expect(cart.isEmpty.value).toBe(true)
  })

  it('setQuantity to zero or below removes the item rather than leaving an invalid quantity', () => {
    const cart = useCart()
    cart.addProduct(makeProduct({ id: 'p1' }))
    cart.setQuantity('p1', 0)
    expect(cart.isEmpty.value).toBe(true)
  })

  it('clear empties the cart', () => {
    const cart = useCart()
    cart.addProduct(makeProduct({ id: 'p1' }))
    cart.addProduct(makeProduct({ id: 'p2' }))
    cart.clear()
    expect(cart.isEmpty.value).toBe(true)
  })
})
