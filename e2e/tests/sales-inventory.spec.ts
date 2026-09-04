import { test, expect } from '@playwright/test'
import { registerAndLoginTestUser, createTestProduct, loginViaUI } from './helpers'

// Deep coverage: Inventory/Stock/Sales is the core of the system. Runs
// against a real backend + Postgres that also
// accumulates data from other test runs and manual verification sessions,
// so every assertion below is scoped to a uniquely-named product/row
// rather than a page-wide text match that could collide with unrelated
// leftover rows.
test.describe('Sales and inventory — deep coverage', () => {
  test('login, create a product, sell it from the POS, stock decrements, and the sale appears in history', async ({
    page,
    request,
  }) => {
    const user = await registerAndLoginTestUser(request)
    await loginViaUI(page, user)

    const productName = `E2E Full Flow ${Date.now()}`

    await page.getByRole('link', { name: 'Productos' }).click()
    await page.getByRole('button', { name: 'Agregar producto' }).click()
    // Scoped to the dialog: ProductsView's own search box ("Buscar
    // productos por nombre") and category filter ("Filtrar por
    // categoría") are still in the DOM behind the modal, and Playwright's
    // getByLabel does a case-insensitive substring match — unscoped, both
    // ambiguously match "Nombre"/"Categoría" too.
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Nombre', { exact: true }).fill(productName)
    await dialog.getByLabel('Precio de venta (MXN)').fill('50')
    await dialog.getByLabel('Costo (MXN)').fill('20')
    await dialog.getByLabel('Stock actual').fill('10')
    await dialog.getByLabel('Stock mínimo').fill('2')
    await dialog.getByRole('button', { name: 'Crear producto' }).click()

    const productRow = page.locator('li', { hasText: productName })
    await expect(productRow).toBeVisible()
    await expect(productRow).toContainText('Stock: 10')

    await page.getByRole('link', { name: 'Punto de venta' }).click()
    await page.getByRole('button', { name: new RegExp(productName) }).click()
    await page.getByRole('button', { name: 'Confirmar venta' }).click()
    await expect(page.getByText('Venta registrada.')).toBeVisible()

    await page.getByRole('link', { name: 'Productos' }).click()
    await expect(page.locator('li', { hasText: productName })).toContainText('Stock: 9')

    await page.getByRole('link', { name: 'Historial de ventas' }).click()
    // Sales are listed most-recent-first and tests run sequentially
    // (workers: 1), so the sale just recorded is the top row.
    await expect(page.locator('.sale-row').first()).toContainText('$50.00')
  })

  test('attempting to oversell shows a clear error and the sale is never completed', async ({ page, request }) => {
    const user = await registerAndLoginTestUser(request)
    const product = await createTestProduct(request, user.token, { currentStock: 1, minStock: 0 })
    await loginViaUI(page, user)

    await page.getByRole('link', { name: 'Punto de venta' }).click()
    const productButton = page.getByRole('button', { name: new RegExp(product.name) })
    await productButton.click()
    await page.getByLabel(`Cantidad de ${product.name}`).fill('2')
    await page.getByRole('button', { name: 'Confirmar venta' }).click()

    await expect(page.locator('.cart-panel .banner--error')).toBeVisible()
    await expect(page.getByText('Venta registrada.')).not.toBeVisible()

    // Stock-never-negative: the failed attempt must not have touched it.
    await page.getByRole('link', { name: 'Productos' }).click()
    await expect(page.locator('li', { hasText: product.name })).toContainText('Stock: 1')
  })

  test('soft-deleting a product with movements sends it to the trash, and a restore rejected for invalid stock shows the backend message', async ({
    page,
    request,
  }) => {
    const user = await registerAndLoginTestUser(request)
    const product = await createTestProduct(request, user.token, { currentStock: 5, minStock: 1 })
    await loginViaUI(page, user)

    // Give the product a real movement (a sale) before deleting it — this
    // is the "product with associated movements" case soft delete exists
    // for: history must survive the delete, never a hard DELETE FROM.
    await page.getByRole('link', { name: 'Punto de venta' }).click()
    await page.getByRole('button', { name: new RegExp(product.name) }).click()
    await page.getByRole('button', { name: 'Confirmar venta' }).click()
    await expect(page.getByText('Venta registrada.')).toBeVisible()

    await page.getByRole('link', { name: 'Productos' }).click()
    page.once('dialog', (dialog) => void dialog.accept())
    await page
      .locator('li', { hasText: product.name })
      .getByRole('button', { name: 'Eliminar' })
      .click()
    await expect(page.locator('li', { hasText: product.name })).not.toBeVisible()

    await page.getByRole('link', { name: 'Ver papelera' }).click()
    await expect(page.locator('li', { hasText: product.name })).toBeVisible()

    // The negative-stock-on-restore rejection is a defended-against
    // invariant that's genuinely unreachable through normal API usage — no
    // endpoint can alter a soft-deleted product's stock, so the backend's
    // own re-validation on restore never actually fails in practice. This
    // test verifies the frontend's handling of that 409 contract via
    // network interception instead of trying to force the real condition,
    // which would mean fighting an invariant the app deliberately upholds.
    await page.route(`**/api/v1/products/${product.id}/restore`, (route) =>
      route.fulfill({
        status: 409,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'cannot restore: current stock is -5, adjust inventory first' }),
      }),
    )
    await page
      .locator('li', { hasText: product.name })
      .getByRole('button', { name: 'Restaurar' })
      .click()

    await expect(page.getByText('cannot restore: current stock is -5, adjust inventory first')).toBeVisible()
    await expect(page.getByText('Ajustá el inventario de este producto antes de restaurarlo.')).toBeVisible()
    // Rejected — it must still be in the trash, not silently restored.
    await expect(page.locator('li', { hasText: product.name })).toBeVisible()
  })

  test('offline: a sale is queued as PENDING_SYNC and syncs automatically on reconnect', async ({
    page,
    request,
    context,
  }) => {
    const user = await registerAndLoginTestUser(request)
    const product = await createTestProduct(request, user.token, { currentStock: 5, minStock: 0 })
    await loginViaUI(page, user)

    // The service worker installs/activates asynchronously after first
    // load — waiting for it explicitly (rather than racing it) is what
    // makes offline navigation to a not-yet-visited lazy route (Historial
    // de ventas, below) reliable: vite-plugin-pwa's generateSW strategy
    // precaches every route's JS chunk at install time, so once this
    // resolves, everything is already available with no network at all.
    await page.evaluate(() => navigator.serviceWorker.ready)

    await page.getByRole('link', { name: 'Punto de venta' }).click()
    // Let the catalog load while still online — the offline flow being
    // tested is "sell while offline", not "browse while offline" (that's
    // the service-worker cache's job, exercised manually per the session
    // checklist, not here).
    await expect(page.getByRole('button', { name: new RegExp(product.name) })).toBeVisible()

    await context.setOffline(true)
    await expect(page.getByText('Sin conexión')).toBeVisible()

    await page.getByRole('button', { name: new RegExp(product.name) }).click()
    await page.getByRole('button', { name: 'Confirmar venta' }).click()
    await expect(page.getByText(/Venta guardada/)).toBeVisible()

    await page.getByRole('link', { name: 'Historial de ventas' }).click()
    // Scoped to the top (most recent) row: this shared dev database has
    // plenty of other confirmed sales already showing "Sincronizada" from
    // earlier tests/sessions, so an unscoped getByText would be ambiguous.
    // Rows are sorted by createdAt across both local and synced sales (see
    // SalesHistoryView's rows computed) — the sale just made here is the
    // newest either way, so it stays the first row before and after sync.
    const topRow = page.locator('.sale-row').first()
    await expect(topRow).toContainText('Pendiente de sincronizar')

    await context.setOffline(false)

    await expect(topRow).toContainText('Sincronizada', { timeout: 15_000 })
  })
})
