import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  // envDir defaults to this project's own root (frontend/brew-ops/.env) —
  // deliberately NOT the repo root or backend/.env, since anything in a
  // Vite .env gets bundled into the public JS. See .env.example.
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'BrewOps',
        short_name: 'BrewOps',
        description: 'Inventario, ventas y marketing para negocios de bebidas.',
        lang: 'es-MX',
        // #29a37a is --color-primary (hsla(160, 60%, 40%, 1)) from
        // tokens.css converted to hex — manifest.json has no hsla() support,
        // so this is the one place in the app a color is expressed any
        // other way. Keep it in sync if --color-primary ever changes.
        theme_color: '#29a37a',
        background_color: '#ffffff',
        display: 'standalone',
        start_url: '/',
        icons: [
          { src: 'favicon.svg', sizes: 'any', type: 'image/svg+xml' },
          { src: 'pwa-192x192.png', sizes: '192x192', type: 'image/png', purpose: 'any maskable' },
          { src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'any maskable' },
        ],
      },
      workbox: {
        runtimeCaching: [
          {
            // The product catalog listing only — not /products/:id,
            // /products/trash, or /products/low-stock, which aren't what
            // "browse the catalog offline" means. Matched by pathname
            // (not full URL) so it works whether the API is same-origin or
            // on a different Cloud Run origin than the frontend.
            //
            // NetworkFirst, not StaleWhileRevalidate: this app is the ONLY
            // writer of its own catalog (create/edit a product happens in
            // this same PWA), so a request immediately following one of
            // those writes must see the fresh result, not a cache entry
            // from before it. StaleWhileRevalidate always resolves the
            // *caller* with the cached response and only refreshes the
            // cache in the background for next time — verified this
            // concretely during Session 9's E2E run: creating a product
            // then immediately opening the POS view showed the pre-existing
            // cached list, missing the product just created. NetworkFirst
            // still satisfies "browsable with no connection" (that's its
            // fallback path) without that staleness risk while online.
            urlPattern: ({ url, request }: { url: URL; request: Request }) =>
              request.method === 'GET' && url.pathname.endsWith('/api/v1/products'),
            handler: 'NetworkFirst',
            options: {
              cacheName: 'brewops-products-catalog',
              cacheableResponse: { statuses: [0, 200] },
              networkTimeoutSeconds: 3,
            },
          },
          {
            // Product/marketing images — wherever they're actually hosted
            // (local disk in dev, GCP Cloud Storage in production), matched
            // by request type rather than a specific origin/path.
            //
            // StaleWhileRevalidate, not NetworkFirst: unlike the catalog
            // rule above, this app never WRITES an image and immediately
            // needs to see its own write reflected — an admin uploads a
            // product photo through a normal form submit/redirect, not an
            // in-place swap the same view has to reflect a beat later. So
            // the staleness risk that ruled out SWR for /products doesn't
            // apply here, and instant-from-cache paint is worth keeping for
            // images the viewer has already seen.
            //
            // What DOES apply here is the same "must not go stale forever"
            // lesson: this repo's seed images get replaced in place (same
            // filename, new bytes — see backend/seed-assets/ and
            // internal/demoseed) on every demo-reset, so the cache key never
            // changes and a long-lived entry (previously 30 days) could
            // outlive many reset cycles, leaving an installed PWA showing a
            // stale image indefinitely even though a plain browser tab (no
            // service worker) sees the new one immediately. A short
            // maxAgeSeconds bounds that: once an entry expires, the next
            // load refetches and repopulates the cache. 1 hour comfortably
            // clears within a single demo-reset cycle (Cloud Scheduler runs
            // it every 6 hours by default — see
            // terraform/variables.tf's demo_reset_schedule) without paying
            // a network round-trip on every single image view.
            urlPattern: ({ request }: { request: Request }) => request.destination === 'image',
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'brewops-product-images',
              cacheableResponse: { statuses: [0, 200] },
              expiration: { maxEntries: 200, maxAgeSeconds: 60 * 60 },
            },
          },
          // Deliberately nothing else: POST/PATCH/DELETE (every write) and
          // /auth/* are never matched above, so they always hit the
          // network — offline writes are IndexedDB's job (useOfflineSalesDb
          // / useSaleSync from Session 7), not the service worker's.
        ],
      },
    }),
  ],
})
