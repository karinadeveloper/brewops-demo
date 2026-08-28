import { watch } from 'vue'
import { useApi, type ApiError } from './useApi'
import { useConnectivity } from './useConnectivity'
import {
  deletePendingSale,
  getAllPendingSales,
  updatePendingSale,
  type PendingSale,
} from './useOfflineSalesDb'

// Module-level so concurrent callers (e.g. the connectivity watcher firing
// again before a previous sync finished) never run two sync passes at once.
let isSyncing = false
let watcherStarted = false

async function syncOne(sale: PendingSale, onBusinessRejection?: (sale: PendingSale, message: string) => void) {
  const { post } = useApi()
  try {
    await post('/sales', { items: sale.items, payment_method: sale.payment_method })
    // Success — the backend now holds the authoritative record, see
    // useOfflineSalesDb.deletePendingSale's doc comment for why the local
    // copy is removed rather than kept around.
    await deletePendingSale(sale.id)
  } catch (err) {
    const apiError = err as Partial<ApiError>
    if (typeof apiError.status === 'number') {
      // A real HTTP response came back — the backend rejected this sale on
      // business grounds (e.g. insufficient stock, per CLAUDE.md's Sync UX
      // contract). Never auto-retried: the user must decide what to do.
      const message = apiError.message || 'La venta fue rechazada por el servidor.'
      await updatePendingSale({ ...sale, status: 'SYNC_ERROR', errorKind: 'business', errorMessage: message })
      onBusinessRejection?.(sale, message)
    } else {
      // fetch() itself threw — a transport failure (network error/timeout),
      // not a decision from the backend. Safe to retry without asking the
      // user anything first.
      await updatePendingSale({
        ...sale,
        status: 'SYNC_ERROR',
        errorKind: 'transport',
        errorMessage: 'No se pudo conectar con el servidor.',
      })
    }
  }
}

// Syncs every PENDING_SYNC sale, oldest first, one at a time — never in
// parallel, so accumulated offline sales for the same product are checked
// against stock in the order they were actually made, and one sale's
// backend-side stock decrement is visible to the next sale's check.
export async function syncPendingSales(
  onBusinessRejection?: (sale: PendingSale, message: string) => void,
): Promise<void> {
  if (isSyncing) {
    return
  }
  isSyncing = true
  try {
    const pending = (await getAllPendingSales())
      .filter((sale) => sale.status === 'PENDING_SYNC')
      .sort((a, b) => a.createdAt.localeCompare(b.createdAt))

    for (const sale of pending) {
      await syncOne(sale, onBusinessRejection)
    }
  } finally {
    isSyncing = false
  }
}

// Retries a single SYNC_ERROR sale on demand (the "Reintentar" button in
// SalesHistoryView) — used for both transport failures (the expected case)
// and, if the user insists, a business rejection too; syncOne's own logic
// decides the outcome either way.
export async function retryPendingSale(
  id: string,
  onBusinessRejection?: (sale: PendingSale, message: string) => void,
): Promise<void> {
  const all = await getAllPendingSales()
  const sale = all.find((s) => s.id === id)
  if (!sale) {
    return
  }
  await syncOne(sale, onBusinessRejection)
}

// Wires automatic sync to connectivity: fires once whenever isOnline
// transitions from false to true. The watcher itself is a singleton (only
// ever registered once) since useConnectivity's isOnline ref is itself a
// module-level singleton — registering it again per call site would just
// mean the same transition triggers multiple redundant sync passes.
let stopWatcher: (() => void) | null = null

export function useSaleSync(onBusinessRejection?: (sale: PendingSale, message: string) => void) {
  const { isOnline } = useConnectivity()

  if (!watcherStarted) {
    watcherStarted = true
    stopWatcher = watch(isOnline, (online, wasOnline) => {
      if (online && !wasOnline) {
        void syncPendingSales(onBusinessRejection)
      }
    })
  }

  return {
    syncNow: () => syncPendingSales(onBusinessRejection),
    retry: (id: string) => retryPendingSale(id, onBusinessRejection),
  }
}

export function __resetSaleSyncForTests() {
  isSyncing = false
  watcherStarted = false
  stopWatcher?.()
  stopWatcher = null
}
