import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createTestingPinia } from '@pinia/testing'
import DashboardView from './DashboardView.vue'
import { formatCentsAsPesos } from '../utils/money'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: vi.fn(), patch: vi.fn(), delete: vi.fn() }),
}))

// Canvas rendering itself is Chart.js's responsibility, not ours (jsdom has
// no real canvas support anyway) — these stubs let assertions target the
// already-transformed data/options DashboardView's children hand to
// vue-chartjs, via a data attribute, without ever touching a canvas.
vi.mock('vue-chartjs', () => ({
  Line: {
    props: ['data', 'options'],
    template: '<div data-testid="line-chart" :data-chart="JSON.stringify(data)" />',
  },
  Bar: {
    props: ['data', 'options'],
    template: '<div data-testid="bar-chart" :data-chart="JSON.stringify(data)" />',
  },
}))

function chartDataOf(el: HTMLElement): { labels: string[]; datasets: { data: number[] }[] } {
  return JSON.parse(el.getAttribute('data-chart') as string) as {
    labels: string[]
    datasets: { data: number[] }[]
  }
}

const okProduct = { id: 'p1', name: 'Jugo de mango', current_stock: 2, min_stock: 10 }
const topProduct = { product_id: 'p1', name: 'Jugo de mango', quantity_sold: 5, revenue_cents: 7500 }
const revenuePoints = [
  { day: '2026-08-27', total_cents: 1000 },
  { day: '2026-08-28', total_cents: 2000 },
]

function mockGetByPath(responses: Record<string, unknown>) {
  getMock.mockImplementation((path: string) => {
    const entry = Object.entries(responses).find(([prefix]) => path.startsWith(prefix))
    if (!entry) {
      return Promise.reject(new Error(`unexpected path: ${path}`))
    }
    return Promise.resolve(entry[1])
  })
}

function buildTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: DashboardView },
      { path: '/products', name: 'products', component: { template: '<div>products</div>' } },
    ],
  })
}

// LowStockAlerts links each alert to /products?edit=<id> via RouterLink,
// which needs a real router instance in context — a plain pinia-only
// render throws inside vue-router's useLink.
async function renderDashboard() {
  const pinia = createTestingPinia({ stubActions: false })
  const router = buildTestRouter()
  await router.push('/')
  await router.isReady()
  return render(DashboardView, { global: { plugins: [pinia, router] } })
}

describe('DashboardView', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('shows the metric cards formatted as MXN pesos and the sales count as a plain number', async () => {
    mockGetByPath({
      '/reports/revenue': revenuePoints,
      '/sales': [{ id: 's1' }, { id: 's2' }],
      '/reports/inventory-value': { total_value_cents: 500000 },
      '/products/low-stock': [],
      '/reports/top-products': [topProduct],
    })

    await renderDashboard()

    await waitFor(() => expect(screen.getByText(formatCentsAsPesos(3000))).toBeInTheDocument())
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText(formatCentsAsPesos(500000))).toBeInTheDocument()
  })

  it('changing the period recomputes revenue, sales count, and top products with a new date range', async () => {
    mockGetByPath({
      '/reports/revenue': revenuePoints,
      '/sales': [],
      '/reports/inventory-value': { total_value_cents: 0 },
      '/products/low-stock': [],
      '/reports/top-products': [],
    })
    await renderDashboard()
    await waitFor(() => expect(screen.getByText(formatCentsAsPesos(3000))).toBeInTheDocument())
    getMock.mockClear()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Este mes' }))

    await waitFor(() => {
      const revenueCalls = getMock.mock.calls.filter(([path]) => (path as string).startsWith('/reports/revenue'))
      expect(revenueCalls.length).toBeGreaterThan(0)
    })
    const [revenuePath] = getMock.mock.calls.find(([path]) => (path as string).startsWith('/reports/revenue')) as [
      string,
    ]
    // "Este mes" starts on the 1st — a different from= than the default
    // "Últimos 7 días" would have produced, proving the period actually
    // drove a fresh request rather than reusing stale params.
    const params = new URLSearchParams(revenuePath.split('?')[1])
    const fromDate = new Date(params.get('from') as string)
    expect(fromDate.getDate()).toBe(1)
  })

  it('shows the positive empty state when there are no low-stock products', async () => {
    mockGetByPath({
      '/reports/revenue': [],
      '/sales': [],
      '/reports/inventory-value': { total_value_cents: 0 },
      '/products/low-stock': [],
      '/reports/top-products': [],
    })

    await renderDashboard()

    await waitFor(() =>
      expect(screen.getByText('Todo el inventario está en buen nivel')).toBeInTheDocument(),
    )
  })

  it('shows the low-stock list when there are alerted products', async () => {
    mockGetByPath({
      '/reports/revenue': [],
      '/sales': [],
      '/reports/inventory-value': { total_value_cents: 0 },
      '/products/low-stock': [okProduct],
      '/reports/top-products': [],
    })

    await renderDashboard()

    await waitFor(() => expect(screen.getByText('Jugo de mango')).toBeInTheDocument())
    expect(screen.getByText('Stock: 2 · mínimo: 10')).toBeInTheDocument()
  })

  it('an error in one section is shown only there while the others load normally', async () => {
    getMock.mockImplementation((path: string) => {
      if (path.startsWith('/reports/top-products')) {
        return Promise.reject(new Error('boom'))
      }
      if (path.startsWith('/reports/revenue')) {
        return Promise.resolve(revenuePoints)
      }
      if (path.startsWith('/sales')) {
        return Promise.resolve([])
      }
      if (path.startsWith('/reports/inventory-value')) {
        return Promise.resolve({ total_value_cents: 1234 })
      }
      if (path.startsWith('/products/low-stock')) {
        return Promise.resolve([])
      }
      return Promise.reject(new Error(`unexpected path: ${path}`))
    })

    await renderDashboard()

    await waitFor(() => expect(screen.getByText(formatCentsAsPesos(3000))).toBeInTheDocument())
    expect(screen.getByText(formatCentsAsPesos(1234))).toBeInTheDocument()
    const topProductsSection = screen.getByRole('region', { name: 'Productos más vendidos' })
    expect(topProductsSection).toHaveTextContent('No se pudo cargar la gráfica')
  })

  it('passes the transformed revenue points to the line chart in pesos, day-labeled', async () => {
    mockGetByPath({
      '/reports/revenue': revenuePoints,
      '/sales': [],
      '/reports/inventory-value': { total_value_cents: 0 },
      '/products/low-stock': [],
      '/reports/top-products': [],
    })

    await renderDashboard()

    const chartEl = await screen.findByTestId('line-chart')
    const data = chartDataOf(chartEl)
    expect(data.labels).toEqual(['27 ago', '28 ago'])
    expect(data.datasets[0].data).toEqual([10, 20])
  })

  it('passes the transformed top-products ranking to the bar chart in pesos', async () => {
    mockGetByPath({
      '/reports/revenue': [],
      '/sales': [],
      '/reports/inventory-value': { total_value_cents: 0 },
      '/products/low-stock': [],
      '/reports/top-products': [topProduct],
    })

    await renderDashboard()

    const chartEl = await screen.findByTestId('bar-chart')
    const data = chartDataOf(chartEl)
    expect(data.labels).toEqual(['Jugo de mango'])
    expect(data.datasets[0].data).toEqual([75])
  })
})
