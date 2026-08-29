import { defineConfig, devices } from '@playwright/test'

// Runs against a real backend + Postgres, not mocks — see CLAUDE.md's E2E
// scope: deep coverage on inventory/stock/sales, basic on login/reports.
// Postgres itself isn't started here (docker compose up -d postgres, from
// the repo root, must already be running) since Playwright's webServer
// option only knows how to wait on an HTTP URL, not a DB healthcheck.
//
// The frontend is served from a production build (`build && preview`),
// not the dev server: vite-plugin-pwa's service worker — the thing the
// offline test actually exercises — is only generated for a real build,
// so testing against `vite dev` would silently skip precaching entirely
// and every offline-navigation assertion would be exercising a code path
// no real installed PWA ever hits.
export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  // Tests share one backend/database, and some deliberately build on
  // state from earlier in the same file (e.g. a product created in one
  // step is sold in the next) — parallel workers would race each other.
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:4173',
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: 'go run ./cmd/server',
      cwd: '../backend',
      url: 'http://localhost:8080/health',
      reuseExistingServer: true,
      timeout: 30_000,
      // backend/.env's CORS_ALLOWED_ORIGINS default (http://localhost:5173,
      // the Vite dev server) doesn't cover the preview server's port — see
      // the comment above on why this suite serves the frontend from a
      // build+preview rather than dev. godotenv only fills in vars that
      // aren't already set in the process environment, so this genuinely
      // overrides the .env default rather than being silently ignored.
      env: { CORS_ALLOWED_ORIGINS: 'http://localhost:4173,http://localhost:5173' },
    },
    {
      command: 'pnpm run build && pnpm run preview -- --port 4173',
      cwd: '../frontend/brew-ops',
      url: 'http://localhost:4173',
      reuseExistingServer: true,
      timeout: 90_000,
    },
  ],
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
