<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import EmptyState from '../components/shared/EmptyState.vue'
import SkeletonList from '../components/shared/SkeletonList.vue'
import StatusBadge from '../components/shared/StatusBadge.vue'
import { useSalesStore, type Sale } from '../stores/sales'
import { useSaleSync } from '../composables/useSaleSync'
import { getAllPendingSales, type PendingSale } from '../composables/useOfflineSalesDb'
import { formatCentsAsPesos } from '../utils/money'

type HistoryRow =
  | { kind: 'synced'; id: string; createdAt: string; totalCents: number; paymentMethod: string }
  | { kind: 'pending' | 'error'; id: string; createdAt: string; totalCents: number; paymentMethod: string; pendingSale: PendingSale }

const salesStore = useSalesStore()
const { retry, syncVersion } = useSaleSync()

const pendingSales = ref<PendingSale[]>([])
const expandedSaleId = ref<string | null>(null)
const saleDetail = ref<Sale | null>(null)
const detailError = ref('')
const retryingId = ref<string | null>(null)

const dateFormatter = new Intl.DateTimeFormat('es-MX', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'America/Mexico_City',
})

async function loadPendingSales() {
  pendingSales.value = await getAllPendingSales()
}

onMounted(() => {
  void salesStore.fetchSales(1)
  void loadPendingSales()
})

// A sync pass can complete in the background — triggered by the app-level
// connectivity watcher (App.vue), not by anything on this page — while
// this view happens to be mounted. Without re-reading IndexedDB and
// refetching confirmed sales here too, a sale that just finished syncing
// would keep showing "Pendiente de sincronizar" until the user navigated
// away and back.
watch(syncVersion, () => {
  void loadPendingSales()
  void salesStore.fetchSales(salesStore.page)
})

function pendingSaleTotalCents(sale: PendingSale): number {
  return sale.items.reduce((sum, item) => sum + item.unit_price_cents * item.quantity, 0)
}

// Local (not-yet-synced) sales surface here even though the backend has
// never heard of them — merged and sorted alongside confirmed sales so the
// owner sees one continuous history, most recent first.
const rows = computed<HistoryRow[]>(() => {
  const synced: HistoryRow[] = salesStore.items.map((sale) => ({
    kind: 'synced',
    id: sale.id,
    createdAt: sale.created_at,
    totalCents: sale.total_cents,
    paymentMethod: sale.payment_method,
  }))
  const local: HistoryRow[] = pendingSales.value.map((sale) => ({
    kind: sale.status === 'SYNC_ERROR' ? 'error' : 'pending',
    id: sale.id,
    createdAt: sale.createdAt,
    totalCents: pendingSaleTotalCents(sale),
    paymentMethod: sale.payment_method,
    pendingSale: sale,
  }))
  return [...local, ...synced].sort((a, b) => b.createdAt.localeCompare(a.createdAt))
})

async function toggleDetail(row: HistoryRow) {
  if (expandedSaleId.value === row.id) {
    expandedSaleId.value = null
    saleDetail.value = null
    return
  }
  expandedSaleId.value = row.id
  detailError.value = ''
  saleDetail.value = null
  if (row.kind !== 'synced') {
    return
  }
  try {
    saleDetail.value = await salesStore.fetchSaleDetail(row.id)
  } catch {
    detailError.value = 'No se pudo cargar el detalle de la venta.'
  }
}

async function handleRetry(id: string) {
  retryingId.value = id
  try {
    await retry(id)
  } finally {
    retryingId.value = null
    await loadPendingSales()
  }
}

function retryLoad() {
  void salesStore.fetchSales(salesStore.page)
}
</script>

<template>
  <main class="history-view">
    <h1>Historial de ventas</h1>

    <SkeletonList
      v-if="salesStore.isLoading"
      :rows="5"
    />

    <EmptyState
      v-else-if="salesStore.loadError && rows.length === 0"
      title="No se pudo cargar el historial"
      message="Ocurrió un error al conectar con el servidor."
    >
      <template #action>
        <button
          type="button"
          class="btn"
          @click="retryLoad"
        >
          Reintentar
        </button>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="rows.length === 0"
      title="Todavía no hay ventas"
      message="Las ventas que registres en el punto de venta aparecerán acá."
    />

    <!-- A failed fetch of confirmed sales must never hide locally-queued
         PENDING_SYNC/SYNC_ERROR sales — those come from IndexedDB, not the
         network, and are exactly what this view needs to keep showing
         while offline. See the loadError && rows.length === 0 guard above:
         this banner covers the case where the fetch failed but there's
         still something (local sales) worth showing underneath it. -->
    <p
      v-if="salesStore.loadError && rows.length > 0"
      class="banner banner--error"
      role="alert"
    >
      No se pudo actualizar el historial desde el servidor — se muestran las ventas guardadas localmente.
      <button
        type="button"
        class="btn-link"
        @click="retryLoad"
      >
        Reintentar
      </button>
    </p>

    <ul
      v-if="rows.length > 0"
      class="sale-list"
    >
      <li
        v-for="row in rows"
        :key="row.id"
        class="sale-row"
      >
        <button
          type="button"
          class="sale-summary"
          @click="toggleDetail(row)"
        >
          <span class="sale-date">{{ dateFormatter.format(new Date(row.createdAt)) }}</span>
          <span class="sale-total">{{ formatCentsAsPesos(row.totalCents) }}</span>
          <StatusBadge
            v-if="row.kind === 'pending'"
            label="Pendiente de sincronizar"
            variant="warning"
          />
          <StatusBadge
            v-else-if="row.kind === 'error'"
            label="Error de sincronización"
            variant="error"
          />
          <StatusBadge
            v-else
            label="Sincronizada"
            variant="success"
          />
        </button>

        <div
          v-if="row.kind === 'error'"
          class="sale-error"
        >
          <p
            class="banner banner--error"
            role="alert"
          >
            {{ row.pendingSale.errorMessage }}
          </p>
          <button
            v-if="row.pendingSale.errorKind === 'transport'"
            type="button"
            class="btn"
            :disabled="retryingId === row.id"
            @click="handleRetry(row.id)"
          >
            {{ retryingId === row.id ? 'Reintentando…' : 'Reintentar' }}
          </button>
          <p
            v-else
            class="sale-error-note"
          >
            Esta venta no se aplicó por un conflicto de negocio — ajustá el inventario o la venta y volvé a
            registrarla manualmente; no se reintenta automáticamente.
          </p>
        </div>

        <div
          v-if="expandedSaleId === row.id"
          class="sale-detail"
        >
          <p
            v-if="detailError"
            class="banner banner--error"
            role="alert"
          >
            {{ detailError }}
          </p>
          <template v-else-if="row.kind === 'synced' && saleDetail">
            <p>Método de pago: {{ saleDetail.payment_method === 'CASH' ? 'Efectivo' : 'Transferencia' }}</p>
            <ul>
              <li
                v-for="(item, index) in saleDetail.items"
                :key="index"
              >
                {{ item.quantity }} × {{ formatCentsAsPesos(item.unit_price_cents) }}
              </li>
            </ul>
          </template>
          <template v-else-if="row.kind !== 'synced'">
            <p>
              Método de pago:
              {{ row.pendingSale.payment_method === 'CASH' ? 'Efectivo' : 'Transferencia' }}
            </p>
            <ul>
              <li
                v-for="(item, index) in row.pendingSale.items"
                :key="index"
              >
                {{ item.quantity }} × {{ formatCentsAsPesos(item.unit_price_cents) }}
              </li>
            </ul>
          </template>
        </div>
      </li>
    </ul>
  </main>
</template>

<style scoped>
.history-view {
  max-width: 960px;
  margin: 0 auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.sale-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.sale-row {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
}

.sale-summary {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  text-align: left;
}

.sale-date {
  flex: 1;
  color: var(--color-text-muted);
  font-size: 0.875rem;
}

.sale-total {
  font-weight: 600;
}

.sale-error {
  margin-top: var(--space-2);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  align-items: flex-start;
}

.sale-error-note {
  font-size: 0.875rem;
  color: var(--color-text-muted);
}

.sale-detail {
  margin-top: var(--space-2);
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
  font-size: 0.9rem;
}

.banner {
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: 0.9rem;
}

.banner--error {
  color: hsla(4, 75%, 30%, 1);
  background: hsla(4, 75%, 50%, 0.1);
}

.btn {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
}

.btn-link {
  border: none;
  background: none;
  color: inherit;
  padding: 0;
  margin-left: var(--space-2);
  text-decoration: underline;
  font: inherit;
}
</style>
