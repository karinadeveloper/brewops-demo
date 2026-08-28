import { defineStore } from 'pinia'
import { useApi } from '../composables/useApi'

export interface SaleItem {
  product_id: string
  quantity: number
  unit_price_cents: number
}

export interface Sale {
  id: string
  items: SaleItem[]
  total_cents: number
  payment_method: 'CASH' | 'TRANSFER'
  created_at: string
  deleted_at?: string | null
}

export interface CreateSaleInput {
  items: SaleItem[]
  payment_method: 'CASH' | 'TRANSFER'
  idempotency_key: string
}

interface SalesState {
  items: Sale[]
  isLoading: boolean
  loadError: boolean
  hasLoadedOnce: boolean
  page: number
  hasMorePages: boolean
  from: string | null
  to: string | null
}

const PAGE_SIZE = 20

export const useSalesStore = defineStore('sales', {
  state: (): SalesState => ({
    items: [],
    isLoading: false,
    loadError: false,
    hasLoadedOnce: false,
    page: 1,
    hasMorePages: false,
    from: null,
    to: null,
  }),

  actions: {
    async fetchSales(page = 1) {
      this.isLoading = true
      this.loadError = false
      try {
        const { get } = useApi()
        const params = new URLSearchParams()
        if (this.from) {
          params.set('from', this.from)
        }
        if (this.to) {
          params.set('to', this.to)
        }
        params.set('page', String(page))
        params.set('page_size', String(PAGE_SIZE))
        const items = await get<Sale[]>(`/sales?${params.toString()}`)
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

    setDateRange(from: string | null, to: string | null) {
      this.from = from
      this.to = to
      void this.fetchSales(1)
    },

    async fetchSaleDetail(id: string): Promise<Sale> {
      const { get } = useApi()
      return get<Sale>(`/sales/${id}`)
    },

    // Creates a sale directly against the backend — used by the POS when
    // online. Offline queuing (IndexedDB) is handled separately in
    // useOfflineSalesDb/useSaleSync, not here, since this store only
    // knows about sales the backend has actually accepted.
    async createSale(input: CreateSaleInput): Promise<Sale> {
      const { post } = useApi()
      const sale = await post<Sale>('/sales', input)
      this.items.unshift(sale)
      return sale
    },
  },
})
