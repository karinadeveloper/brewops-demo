import { ref } from 'vue'

// Module-level singleton: every caller shares the same isOnline value and
// the same ping loop, rather than each component starting its own
// interval. This is deliberately the ONLY connectivity source in the app —
// the POS and sale-sync logic must consume this, never re-implement their
// own online/offline detection.
const DEFAULT_PING_INTERVAL_MS = 20_000
const PING_TIMEOUT_MS = 3_000

const isOnline = ref(typeof navigator === 'undefined' ? true : navigator.onLine)
let started = false
let intervalId: ReturnType<typeof setInterval> | null = null

// /health lives outside /api/v1 — its URL is derived from
// VITE_API_BASE_URL rather than hardcoding a second base URL to keep in
// sync.
function healthUrl(): string {
  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL as string
  return `${apiBaseUrl.replace(/\/api\/v1\/?$/, '')}/health`
}

// navigator.onLine alone is unreliable — it only reflects whether the
// network interface is up, not whether the backend is actually reachable
// (a captive portal or a dead access point can leave it true while nothing
// loads). This ping is the real check.
async function pingHealth(): Promise<boolean> {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), PING_TIMEOUT_MS)
  try {
    const response = await fetch(healthUrl(), { signal: controller.signal })
    return response.ok
  } catch {
    return false
  } finally {
    clearTimeout(timeoutId)
  }
}

async function runPing() {
  isOnline.value = await pingHealth()
}

function handleOnlineEvent() {
  // The browser says we're back — ping right away instead of waiting for
  // the next interval tick, so the app reacts fast to the obvious case.
  void runPing()
}

function handleOfflineEvent() {
  isOnline.value = false
}

function start(intervalMs: number) {
  if (started) {
    return
  }
  started = true
  window.addEventListener('online', handleOnlineEvent)
  window.addEventListener('offline', handleOfflineEvent)
  void runPing()
  intervalId = setInterval(() => void runPing(), intervalMs)
}

// useConnectivity is a singleton by design: the first call starts the
// listeners/ping loop (module-scoped, not tied to any component's
// lifecycle, since connectivity monitoring is an app-wide concern), every
// later call just returns the same reactive ref.
export function useConnectivity(intervalMs = DEFAULT_PING_INTERVAL_MS) {
  start(intervalMs)
  return { isOnline }
}

// Test-only escape hatch: production code never calls this. Vitest's
// per-file module isolation already gives every test *file* a fresh
// singleton; this lets individual tests within one file reset it too.
export function __resetConnectivityForTests() {
  started = false
  if (intervalId !== null) {
    clearInterval(intervalId)
    intervalId = null
  }
  window.removeEventListener('online', handleOnlineEvent)
  window.removeEventListener('offline', handleOfflineEvent)
  isOnline.value = typeof navigator === 'undefined' ? true : navigator.onLine
}
