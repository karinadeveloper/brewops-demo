import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  // Read .env from the repo root so the frontend shares a single
  // .env / .env.example with the backend instead of duplicating it.
  envDir: '../../',
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      // Full offline caching strategy (product catalog, PENDING_SYNC sales)
      // is implemented in Session 9 — this is the base manifest only.
      manifest: {
        name: 'BrewOps',
        short_name: 'BrewOps',
        description: 'Inventario, ventas y marketing para negocios de bebidas.',
        theme_color: '#1f8f6b',
        background_color: '#ffffff',
        display: 'standalone',
        start_url: '/',
        icons: [],
      },
    }),
  ],
})
