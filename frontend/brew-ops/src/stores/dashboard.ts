import { defineStore } from 'pinia'
import { useApi } from '../composables/useApi'
import type { Product } from './products'
import { resolvePeriodRange, type Period } from '../utils/period'

export interface RevenuePoint {
  day: string | null
  total_cents: number
}

export interface TopProduct {
  product_id: string
  name: string
  quantity_sold: number
  revenue_cents: number
}

// GET /sales has no endpoint returning just a count, so the "sales this
// period" card paginates through it and sums page lengths. A page size of
// 100 with this cap bounds the worst case at 2000 sales per period — far
// beyond what a small juice/beverage business does in a day, a week, or a
// month — while still producing an exact count instead of guessing from a
// single oversized page.
const SALES_COUNT_PAGE_SIZE = 100
const SALES_COUNT_MAX_PAGES = 20

interface DashboardState {
  period: Period

  revenuePoints: RevenuePoint[]
  revenueTotalCents: number
  isLoadingRevenue: boolean
  revenueError: boolean

  salesCount: number
  isLoadingSalesCount: boolean
  salesCountError: boolean

  // Inventory value and low-stock reflect the current moment, not the
  // selected period — see loadAll/setPeriod below for why they aren't
  // refetched when the period changes.
  inventoryValueCents: number
  isLoadingInventoryValue: boolean
  inventoryValueError: boolean

  lowStockItems: Product[]
  isLoadingLowStock: boolean
  lowStockError: boolean

  topProducts: TopProduct[]
  isLoadingTopProducts: boolean
  topProductsError: boolean
}

export const useDashboardStore = defineStore('dashboard', {
  state: (): DashboardState => ({
    period: 'last7',

    revenuePoints: [],
    revenueTotalCents: 0,
    isLoadingRevenue: false,
    revenueError: false,

    salesCount: 0,
    isLoadingSalesCount: false,
    salesCountError: false,

    inventoryValueCents: 0,
    isLoadingInventoryValue: false,
    inventoryValueError: false,

    lowStockItems: [],
    isLoadingLowStock: false,
    lowStockError: false,

    topProducts: [],
    isLoadingTopProducts: false,
    topProductsError: false,
  }),

  actions: {
    // Loads every section once, each independently, so a failure in one
    // (e.g. inventory value) never blocks or hides the others — each
    // section tracks and surfaces its own loading/error state.
    async loadAll() {
      await Promise.all([this.loadPeriodDependent(), this.loadInventoryValue(), this.loadLowStock()])
    },

    // Only revenue, sales count, and top products are computed over the
    // selected date range — current inventory value and low-stock alerts
    // describe right now, not a historical window, so changing the period
    // intentionally does not refetch them.
    setPeriod(period: Period) {
      this.period = period
      void this.loadPeriodDependent()
    },

    async loadPeriodDependent() {
      await Promise.all([this.loadRevenue(), this.loadSalesCount(), this.loadTopProducts()])
    },

    async loadRevenue() {
      this.isLoadingRevenue = true
      this.revenueError = false
      try {
        const { get } = useApi()
        const { from, to } = resolvePeriodRange(this.period)
        const params = new URLSearchParams({ from, to, group_by: 'day' })
        const points = await get<RevenuePoint[]>(`/reports/revenue?${params.toString()}`)
        this.revenuePoints = points
        this.revenueTotalCents = points.reduce((sum, p) => sum + p.total_cents, 0)
      } catch {
        this.revenueError = true
      } finally {
        this.isLoadingRevenue = false
      }
    },

    async loadSalesCount() {
      this.isLoadingSalesCount = true
      this.salesCountError = false
      try {
        const { get } = useApi()
        const { from, to } = resolvePeriodRange(this.period)
        let count = 0
        for (let page = 1; page <= SALES_COUNT_MAX_PAGES; page++) {
          const params = new URLSearchParams({
            from,
            to,
            page: String(page),
            page_size: String(SALES_COUNT_PAGE_SIZE),
          })
          const items = await get<unknown[]>(`/sales?${params.toString()}`)
          count += items.length
          if (items.length < SALES_COUNT_PAGE_SIZE) {
            break
          }
        }
        this.salesCount = count
      } catch {
        this.salesCountError = true
      } finally {
        this.isLoadingSalesCount = false
      }
    },

    async loadInventoryValue() {
      this.isLoadingInventoryValue = true
      this.inventoryValueError = false
      try {
        const { get } = useApi()
        const resp = await get<{ total_value_cents: number }>('/reports/inventory-value')
        this.inventoryValueCents = resp.total_value_cents
      } catch {
        this.inventoryValueError = true
      } finally {
        this.isLoadingInventoryValue = false
      }
    },

    async loadLowStock() {
      this.isLoadingLowStock = true
      this.lowStockError = false
      try {
        const { get } = useApi()
        this.lowStockItems = await get<Product[]>('/products/low-stock')
      } catch {
        this.lowStockError = true
      } finally {
        this.isLoadingLowStock = false
      }
    },

    async loadTopProducts() {
      this.isLoadingTopProducts = true
      this.topProductsError = false
      try {
        const { get } = useApi()
        const { from, to } = resolvePeriodRange(this.period)
        const params = new URLSearchParams({ from, to })
        this.topProducts = await get<TopProduct[]>(`/reports/top-products?${params.toString()}`)
      } catch {
        this.topProductsError = true
      } finally {
        this.isLoadingTopProducts = false
      }
    },
  },
})
