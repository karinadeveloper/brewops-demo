import { test, expect } from '@playwright/test'
import { registerAndLoginTestUser } from './helpers'

// Basic coverage per CLAUDE.md's E2E scope: Login and Reports are
// lower-risk, standard CRUD/read flows — happy path plus 1-2 error cases.
test.describe('Login and reports — basic coverage', () => {
  test('a successful login lands on the dashboard', async ({ page, request }) => {
    const user = await registerAndLoginTestUser(request)

    await page.goto('/login')
    await page.getByLabel('Correo electrónico').fill(user.email)
    await page.getByLabel('Contraseña').fill(user.password)
    await page.getByRole('button', { name: 'Iniciar sesión' }).click()

    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  })

  test('an incorrect password shows an error and stays on the login page', async ({ page, request }) => {
    const user = await registerAndLoginTestUser(request)

    await page.goto('/login')
    await page.getByLabel('Correo electrónico').fill(user.email)
    await page.getByLabel('Contraseña').fill('wrong-password')
    await page.getByRole('button', { name: 'Iniciar sesión' }).click()

    await expect(page.getByText('Email o contraseña incorrectos.')).toBeVisible()
    await expect(page).toHaveURL(/\/login/)
  })

  test('the dashboard loads its metric cards and charts without a visible error', async ({ page, request }) => {
    const user = await registerAndLoginTestUser(request)

    await page.goto('/login')
    await page.getByLabel('Correo electrónico').fill(user.email)
    await page.getByLabel('Contraseña').fill(user.password)
    await page.getByRole('button', { name: 'Iniciar sesión' }).click()
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()

    // A fresh user has no products/sales yet, so this deliberately doesn't
    // assert on data values — only that every section resolved to a real
    // state (a value or an intentional empty state), never its error
    // branch ("No se pudo cargar...").
    await expect(page.locator('.metric-card').first()).toBeVisible()
    await expect(page.getByText('No se pudo cargar', { exact: false })).not.toBeVisible()
    await expect(page.locator('canvas').first()).toBeVisible({ timeout: 10_000 })
  })
})
