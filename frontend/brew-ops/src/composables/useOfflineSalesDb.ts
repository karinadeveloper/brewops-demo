// Thin promise wrapper around the raw IndexedDB API — hand-rolled rather
// than pulling in a dependency (e.g. idb) for what's ultimately a single
// object store with four operations, consistent with this project's
// preference for demonstrating the underlying platform API directly (see
// e.g. the backend's hand-rolled OpenAI HTTP client).

const DB_NAME = 'brewops-offline'
const DB_VERSION = 1
const STORE_NAME = 'pending_sales'

export type PendingSaleStatus = 'PENDING_SYNC' | 'SYNC_ERROR'
// "business": the backend responded and rejected the sale on business
// grounds (e.g. insufficient stock) — never auto-retried, the user must
// resolve it. "transport": the request never got a response at all
// (network error/timeout) — safe to retry blindly. See CLAUDE.md's Sync
// UX contract.
export type SyncErrorKind = 'business' | 'transport'

export interface PendingSaleItem {
  product_id: string
  quantity: number
  unit_price_cents: number
}

export interface PendingSale {
  id: string
  items: PendingSaleItem[]
  payment_method: 'CASH' | 'TRANSFER'
  // Local creation timestamp (ISO string) — used to sync in creation
  // order and to display in the history list.
  createdAt: string
  status: PendingSaleStatus
  errorMessage?: string
  errorKind?: SyncErrorKind
}

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: 'id' })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error as Error)
  })
}

async function withStore<T>(
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  const db = await openDb()
  try {
    return await new Promise<T>((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, mode)
      const request = run(transaction.objectStore(STORE_NAME))
      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error as Error)
    })
  } finally {
    db.close()
  }
}

export async function addPendingSale(sale: PendingSale): Promise<void> {
  await withStore('readwrite', (store) => store.add(sale))
}

export async function getAllPendingSales(): Promise<PendingSale[]> {
  return withStore('readonly', (store) => store.getAll())
}

export async function updatePendingSale(sale: PendingSale): Promise<void> {
  await withStore('readwrite', (store) => store.put(sale))
}

// Deliberately deletes rather than keeping a "synced" record around: once
// a sale is confirmed against the backend, the authoritative copy lives in
// Postgres and is retrievable via GET /sales — keeping a duplicate local
// copy would only let the local store grow unbounded and add "is this the
// same sale as that one" ambiguity to the history view. Local storage is
// reserved for sales that still need attention (PENDING_SYNC/SYNC_ERROR).
export async function deletePendingSale(id: string): Promise<void> {
  await withStore('readwrite', (store) => store.delete(id))
}
