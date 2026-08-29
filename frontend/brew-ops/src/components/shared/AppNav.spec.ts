import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { render, screen } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createTestingPinia } from '@pinia/testing'
import AppNav from './AppNav.vue'
import { useAuthStore } from '../../stores/auth'

const StubView = { template: '<div />' }

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: StubView },
      { path: '/pos', name: 'pos', component: StubView },
      { path: '/sales', name: 'sales-history', component: StubView },
      { path: '/products', name: 'products', component: StubView },
      { path: '/products/trash', name: 'products-trash', component: StubView },
      { path: '/marketing', name: 'marketing', component: StubView },
    ],
  })
}

async function renderNav(initialPath = '/') {
  const router = makeRouter()
  await router.push(initialPath)
  await router.isReady()
  // Default createTestingPinia stubs actions (logout included), so the
  // test can assert it was called without it running the real
  // localStorage/router side effects.
  const pinia = createTestingPinia()
  const utils = render(AppNav, { global: { plugins: [pinia, router] } })
  return { router, pinia, ...utils }
}

describe('AppNav', () => {
  it('renders a link to every main section', async () => {
    await renderNav()

    expect(screen.getByRole('link', { name: 'Dashboard' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Punto de venta' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Historial de ventas' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Productos' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Marketing' })).toBeInTheDocument()
  })

  it('marks the current route as active with aria-current, and no other link', async () => {
    await renderNav('/products')

    expect(screen.getByRole('link', { name: 'Productos' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Dashboard' })).not.toHaveAttribute('aria-current')
    expect(screen.getByRole('link', { name: 'Marketing' })).not.toHaveAttribute('aria-current')
  })

  it('the root route is only active on "/", not on every other route', async () => {
    await renderNav('/pos')

    expect(screen.getByRole('link', { name: 'Punto de venta' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Dashboard' })).not.toHaveAttribute('aria-current')
  })

  it('clicking "Cerrar sesión" calls the auth store logout action', async () => {
    const { pinia } = await renderNav()
    const authStore = useAuthStore(pinia)
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Cerrar sesión' }))

    expect(authStore.logout).toHaveBeenCalledOnce()
  })

  it('the mobile menu toggle starts closed and opens on click', async () => {
    await renderNav()
    const toggle = screen.getByRole('button', { name: 'Abrir menú de navegación' })

    expect(toggle).toHaveAttribute('aria-expanded', 'false')

    const user = userEvent.setup()
    await user.click(toggle)

    expect(toggle).toHaveAttribute('aria-expanded', 'true')
  })

  it('clicking a link closes an open mobile menu', async () => {
    await renderNav()
    const toggle = screen.getByRole('button', { name: 'Abrir menú de navegación' })
    const user = userEvent.setup()
    await user.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')

    await user.click(screen.getByRole('link', { name: 'Productos' }))

    expect(toggle).toHaveAttribute('aria-expanded', 'false')
  })
})
