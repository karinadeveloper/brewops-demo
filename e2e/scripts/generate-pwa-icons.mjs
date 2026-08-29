// One-off script, not part of any build pipeline: renders
// frontend/brew-ops/public/favicon.svg into the raster PWA icon sizes
// Chrome's installability check expects (192x192, 512x512). No
// image-conversion CLI (ImageMagick, rsvg-convert, sharp) was available in
// this environment, but Playwright's bundled Chromium — already a
// dependency here for E2E — can screenshot an SVG just as well. Lives in
// e2e/ (not frontend/brew-ops/) so the frontend package doesn't need
// Playwright as a dependency just for this. Run again if favicon.svg's
// design ever changes:
//
//   cd e2e && node scripts/generate-pwa-icons.mjs
import { chromium } from '@playwright/test'
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const publicDir = path.resolve(__dirname, '..', '..', 'frontend', 'brew-ops', 'public')
const svg = readFileSync(path.join(publicDir, 'favicon.svg'), 'utf-8')

// Matches manifest.background_color in vite.config.ts — the icon must look
// intentional on the home screen, not like a sticker on a mismatched tile.
const BACKGROUND = '#ffffff'

const sizes = [192, 512]

const browser = await chromium.launch()
const page = await browser.newPage()

for (const size of sizes) {
  await page.setViewportSize({ width: size, height: size })
  const iconSize = Math.round(size * 0.7)
  // The source SVG's own viewBox is 48x46 — centered with padding so the
  // mark doesn't touch the edges once scaled up to icon size.
  await page.setContent(`
    <html>
      <body style="margin:0;width:${size}px;height:${size}px;background:${BACKGROUND};display:flex;align-items:center;justify-content:center;">
        ${svg}
      </body>
    </html>
  `)
  await page.locator('svg').evaluate((el, s) => {
    el.setAttribute('width', String(s))
    el.setAttribute('height', String(s))
  }, iconSize)
  const buffer = await page.screenshot({ omitBackground: false })
  writeFileSync(path.join(publicDir, `pwa-${size}x${size}.png`), buffer)
  console.log(`wrote frontend/brew-ops/public/pwa-${size}x${size}.png`)
}

await browser.close()
