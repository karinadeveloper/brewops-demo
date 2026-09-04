import type { APIRequestContext, Page } from '@playwright/test'
import { expect } from '@playwright/test'

const API_BASE = 'http://localhost:8080/api/v1'

export interface TestUser {
  email: string
  password: string
  token: string
}

// POST /auth/register is dev-only (404s outside APP_ENV=development) — used
// here purely to give each test run its own isolated user, so tests never
// collide with data left behind by a previous run or another test file.
export async function registerAndLoginTestUser(request: APIRequestContext): Promise<TestUser> {
  const email = `e2e-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@brewops.mx`
  const password = 'testpass123'

  await request.post(`${API_BASE}/auth/register`, { data: { email, password } })
  const loginResp = await request.post(`${API_BASE}/auth/login`, { data: { email, password } })
  const { access_token: token } = (await loginResp.json()) as { access_token: string }

  return { email, password, token }
}

// Seeds a product directly via the API — used when a test's focus is
// somewhere else (a sale, a restore) and walking through the create-product
// UI would just be setup noise. The one test that specifically covers
// product creation drives ProductsView's form for real instead.
export async function createTestProduct(
  request: APIRequestContext,
  token: string,
  overrides: {
    name?: string
    currentStock?: number
    minStock?: number
    salePriceCents?: number
  } = {},
): Promise<{ id: string; name: string }> {
  const name = overrides.name ?? `E2E Product ${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
  const resp = await request.post(`${API_BASE}/products`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      name,
      category: 'juice',
      sale_price_cents: overrides.salePriceCents ?? 5000,
      cost_cents: 2000,
      current_stock: overrides.currentStock ?? 10,
      min_stock: overrides.minStock ?? 2,
    },
  })
  const body = (await resp.json()) as { id: string; name: string }
  return { id: body.id, name: body.name }
}

// Logs in through the real LoginView form (not an API shortcut) — every
// test that needs an authenticated session starts here, since the point of
// an E2E suite is to exercise the actual login flow the owner uses.
export async function loginViaUI(page: Page, user: Pick<TestUser, 'email' | 'password'>) {
  await page.goto('/login')
  await page.getByLabel('Correo electrónico').fill(user.email)
  await page.getByLabel('Contraseña').fill(user.password)
  await page.getByRole('button', { name: 'Iniciar sesión' }).click()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
}
