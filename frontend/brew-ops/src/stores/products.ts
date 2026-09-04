import { defineStore } from 'pinia'
import { useApi } from '../composables/useApi'

export type ProductCategory = 'juice' | 'water' | 'soda' | 'other'

export interface Product {
  id: string
  name: string
  category: ProductCategory
  sale_price_cents: number
  cost_cents: number
  current_stock: number
  min_stock: number
  image_url: string | null
  version: number
  created_at: string
  updated_at: string
  deleted_at?: string | null
  deleted_by_email?: string | null
}

export interface ProductInput {
  name: string
  category: ProductCategory
  sale_price_cents: number
  cost_cents: number
  current_stock: number
  min_stock: number
  image_url: string | null
}

const PAGE_SIZE = 20

interface ProductsState {
  items: Product[]
  trashedItems: Product[]
  isLoading: boolean
  loadError: boolean
  // Distinguishes "never fetched yet" from "fetched and genuinely empty" —
  // needed to tell apart the "no products at all" empty state from a
  // loading flash on first render.
  hasLoadedOnce: boolean
  isLoadingTrash: boolean
  trashLoadError: boolean
  categoryFilter: ProductCategory | null
  searchQuery: string
  page: number
  // Not driven by a total count from the backend (GET /products doesn't
  // return one) — true whenever the last fetched page came back full,
  // meaning there might be more. See fetchProducts.
  hasMorePages: boolean
}

export const useProductsStore = defineStore('products', {
  state: (): ProductsState => ({
    items: [],
    trashedItems: [],
    isLoading: false,
    loadError: false,
    hasLoadedOnce: false,
    isLoadingTrash: false,
    trashLoadError: false,
    categoryFilter: null,
    searchQuery: '',
    page: 1,
    hasMorePages: false,
  }),

  getters: {
    // Client-side text search — filters what's already loaded, no new
    // endpoint. Category filtering itself is sent to the backend as a
    // query param (fetchProducts), not duplicated here.
    filteredItems(state): Product[] {
      const query = state.searchQuery.trim().toLowerCase()
      if (!query) {
        return state.items
      }
      return state.items.filter((product) => product.name.toLowerCase().includes(query))
    },
  },

  actions: {
    async fetchProducts(page = 1) {
      this.isLoading = true
      this.loadError = false
      try {
        const { get } = useApi()
        const params = new URLSearchParams()
        if (this.categoryFilter) {
          params.set('category', this.categoryFilter)
        }
        params.set('page', String(page))
        params.set('page_size', String(PAGE_SIZE))
        const items = await get<Product[]>(`/products?${params.toString()}`)
        this.items = items
        this.page = page
        this.hasMorePages = items.length === PAGE_SIZE
      } catch {
        this.loadError = true
      } finally {
        this.isLoading = false
        this.hasLoadedOnce = true
      }
    },

    setCategoryFilter(category: ProductCategory | null) {
      this.categoryFilter = category
      void this.fetchProducts(1)
    },

    // Fetches a single product by id regardless of pagination/filter state
    // — used by ProductsView to open the edit modal for a product deep-linked
    // from the Dashboard's low-stock alerts, which may not be on the
    // currently-loaded page.
    async getProduct(id: string): Promise<Product> {
      const { get } = useApi()
      return get<Product>(`/products/${id}`)
    },

    async createProduct(input: ProductInput): Promise<Product> {
      const { post } = useApi()
      const product = await post<Product>('/products', input)
      this.items.unshift(product)
      return product
    },

    async updateProduct(id: string, version: number, input: ProductInput): Promise<Product> {
      const { patch } = useApi()
      const updated = await patch<Product>(`/products/${id}`, { ...input, version })
      const index = this.items.findIndex((p) => p.id === id)
      if (index !== -1) {
        this.items[index] = updated
      }
      return updated
    },

    async softDeleteProduct(id: string) {
      const { delete: del } = useApi()
      await del(`/products/${id}`)
      this.items = this.items.filter((p) => p.id !== id)
    },

    async fetchTrash() {
      this.isLoadingTrash = true
      this.trashLoadError = false
      try {
        const { get } = useApi()
        this.trashedItems = await get<Product[]>('/products/trash')
      } catch {
        this.trashLoadError = true
      } finally {
        this.isLoadingTrash = false
      }
    },

    async restoreProduct(id: string): Promise<Product> {
      const { post } = useApi()
      const restored = await post<Product>(`/products/${id}/restore`)
      this.trashedItems = this.trashedItems.filter((p) => p.id !== id)
      return restored
    },
  },
})
