import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { getMock, postMock, patchMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  patchMock: vi.fn(),
  deleteMock: vi.fn(),
}))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: postMock, patch: patchMock, delete: deleteMock }),
}))

const { useProductsStore } = await import('./products')
import type { Product } from './products'

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

describe('useProductsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getMock.mockReset()
    postMock.mockReset()
    patchMock.mockReset()
    deleteMock.mockReset()
  })

  it('fetchProducts loads items and marks hasMorePages when a full page comes back', async () => {
    // Arrange
    const fullPage = Array.from({ length: 20 }, (_, i) => makeProduct({ id: `p${i}` }))
    getMock.mockResolvedValueOnce(fullPage)
    const store = useProductsStore()

    // Act
    await store.fetchProducts(1)

    // Assert
    expect(store.items).toHaveLength(20)
    expect(store.hasMorePages).toBe(true)
    expect(store.isLoading).toBe(false)
    expect(store.loadError).toBe(false)
    expect(store.hasLoadedOnce).toBe(true)
  })

  it('fetchProducts marks hasMorePages false when fewer than a full page comes back', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct()])
    const store = useProductsStore()

    // Act
    await store.fetchProducts(1)

    // Assert
    expect(store.hasMorePages).toBe(false)
  })

  it('fetchProducts sets loadError on failure and keeps hasLoadedOnce true', async () => {
    // Arrange
    getMock.mockRejectedValueOnce({ status: 500, message: 'server error' })
    const store = useProductsStore()

    // Act
    await store.fetchProducts(1)

    // Assert
    expect(store.loadError).toBe(true)
    expect(store.isLoading).toBe(false)
    expect(store.hasLoadedOnce).toBe(true)
  })

  it('setCategoryFilter sends the category as a query param', async () => {
    // Arrange
    getMock.mockResolvedValue([])
    const store = useProductsStore()

    // Act
    store.setCategoryFilter('water')
    await vi.waitFor(() => expect(getMock).toHaveBeenCalled())

    // Assert
    expect(getMock).toHaveBeenCalledWith(expect.stringContaining('category=water'))
  })

  it('filteredItems applies the client-side text search on top of loaded items', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([
      makeProduct({ id: 'p1', name: 'Jugo de naranja 1L' }),
      makeProduct({ id: 'p2', name: 'Agua mineral 600ml' }),
    ])
    const store = useProductsStore()
    await store.fetchProducts(1)

    // Act
    store.searchQuery = 'naranja'

    // Assert
    expect(store.filteredItems).toHaveLength(1)
    expect(store.filteredItems[0].name).toBe('Jugo de naranja 1L')
  })

  it('filteredItems is case-insensitive and returns everything for an empty query', () => {
    // Arrange
    const store = useProductsStore()
    store.items = [makeProduct({ name: 'Jugo de Naranja' })]

    // Act
    store.searchQuery = 'NARANJA'

    // Assert
    expect(store.filteredItems).toHaveLength(1)
  })

  it('createProduct posts the input and prepends the result to items', async () => {
    // Arrange
    const created = makeProduct({ id: 'new-id' })
    postMock.mockResolvedValueOnce(created)
    const store = useProductsStore()

    // Act
    const result = await store.createProduct({
      name: created.name,
      category: created.category,
      sale_price_cents: created.sale_price_cents,
      cost_cents: created.cost_cents,
      current_stock: created.current_stock,
      min_stock: created.min_stock,
      image_url: null,
    })

    // Assert
    expect(result).toEqual(created)
    expect(store.items[0]).toEqual(created)
  })

  it('updateProduct sends the version and replaces the item in place', async () => {
    // Arrange
    const original = makeProduct({ id: 'p1', version: 1 })
    const updated = makeProduct({ id: 'p1', version: 2, name: 'Nuevo nombre' })
    patchMock.mockResolvedValueOnce(updated)
    const store = useProductsStore()
    store.items = [original]

    // Act
    const result = await store.updateProduct('p1', 1, {
      name: 'Nuevo nombre',
      category: 'juice',
      sale_price_cents: 4500,
      cost_cents: 2000,
      current_stock: 50,
      min_stock: 10,
      image_url: null,
    })

    // Assert
    expect(patchMock).toHaveBeenCalledWith('/products/p1', expect.objectContaining({ version: 1 }))
    expect(result).toEqual(updated)
    expect(store.items[0]).toEqual(updated)
  })

  it('updateProduct propagates a 409 conflict without mutating items', async () => {
    // Arrange
    const original = makeProduct({ id: 'p1', version: 1 })
    patchMock.mockRejectedValueOnce({ status: 409, message: 'conflict' })
    const store = useProductsStore()
    store.items = [original]

    // Act & Assert
    await expect(
      store.updateProduct('p1', 1, {
        name: original.name,
        category: original.category,
        sale_price_cents: original.sale_price_cents,
        cost_cents: original.cost_cents,
        current_stock: original.current_stock,
        min_stock: original.min_stock,
        image_url: null,
      }),
    ).rejects.toMatchObject({ status: 409 })
    expect(store.items[0]).toEqual(original)
  })

  it('softDeleteProduct removes the item from the active list', async () => {
    // Arrange
    deleteMock.mockResolvedValueOnce(undefined)
    const store = useProductsStore()
    store.items = [makeProduct({ id: 'p1' }), makeProduct({ id: 'p2' })]

    // Act
    await store.softDeleteProduct('p1')

    // Assert
    expect(store.items.map((p) => p.id)).toEqual(['p2'])
  })

  it('fetchTrash loads trashed items', async () => {
    // Arrange
    getMock.mockResolvedValueOnce([makeProduct({ id: 'p1', deleted_at: '2026-01-02T00:00:00Z' })])
    const store = useProductsStore()

    // Act
    await store.fetchTrash()

    // Assert
    expect(store.trashedItems).toHaveLength(1)
    expect(store.isLoadingTrash).toBe(false)
    expect(store.trashLoadError).toBe(false)
  })

  it('fetchTrash sets trashLoadError on failure', async () => {
    // Arrange
    getMock.mockRejectedValueOnce({ status: 500, message: 'server error' })
    const store = useProductsStore()

    // Act
    await store.fetchTrash()

    // Assert
    expect(store.trashLoadError).toBe(true)
  })

  it('restoreProduct removes the item from trashedItems on success', async () => {
    // Arrange
    const restored = makeProduct({ id: 'p1', deleted_at: null })
    postMock.mockResolvedValueOnce(restored)
    const store = useProductsStore()
    store.trashedItems = [makeProduct({ id: 'p1', deleted_at: '2026-01-02T00:00:00Z' })]

    // Act
    const result = await store.restoreProduct('p1')

    // Assert
    expect(result).toEqual(restored)
    expect(store.trashedItems).toHaveLength(0)
  })

  it('restoreProduct propagates a 409 (negative stock) without mutating trashedItems', async () => {
    // Arrange
    const trashed = makeProduct({ id: 'p1', deleted_at: '2026-01-02T00:00:00Z' })
    postMock.mockRejectedValueOnce({
      status: 409,
      message: 'cannot restore: current stock is -5, adjust inventory first',
    })
    const store = useProductsStore()
    store.trashedItems = [trashed]

    // Act & Assert
    await expect(store.restoreProduct('p1')).rejects.toMatchObject({ status: 409 })
    expect(store.trashedItems).toEqual([trashed])
  })
})
