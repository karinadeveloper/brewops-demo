<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { RouterLink } from 'vue-router'
import EmptyState from '../components/shared/EmptyState.vue'
import SkeletonList from '../components/shared/SkeletonList.vue'
import { useProductsStore } from '../stores/products'
import { formatCentsAsPesos } from '../utils/money'
import type { ApiError } from '../composables/useApi'

const productsStore = useProductsStore()

// Keyed by product id — restoring one product failing (e.g. negative
// stock) shouldn't affect the message shown for any other row.
const restoreErrors = reactive<Record<string, string>>({})

const dateFormatter = new Intl.DateTimeFormat('es-MX', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'America/Mexico_City',
})

onMounted(() => {
  void productsStore.fetchTrash()
})

async function handleRestore(id: string) {
  delete restoreErrors[id]
  try {
    await productsStore.restoreProduct(id)
  } catch (err) {
    const apiError = err as ApiError
    if (apiError.status === 409) {
      // Per CLAUDE.md's restore-revalidation rule: the backend's message
      // is already descriptive ("cannot restore: current stock is -5,
      // adjust inventory first") — shown verbatim, plus our own suggested
      // next step. No link to the inventory movements view yet since it
      // doesn't exist as of this session.
      restoreErrors[id] = apiError.message
    } else {
      restoreErrors[id] = 'No se pudo restaurar el producto. Intentá de nuevo.'
    }
  }
}

function retryLoad() {
  void productsStore.fetchTrash()
}
</script>

<template>
  <main class="trash-view">
    <header class="trash-header">
      <h1>Papelera de productos</h1>
      <RouterLink
        to="/products"
        class="btn"
      >
        Volver a productos
      </RouterLink>
    </header>

    <SkeletonList
      v-if="productsStore.isLoadingTrash"
      :rows="4"
    />

    <EmptyState
      v-else-if="productsStore.trashLoadError"
      title="No se pudo cargar la papelera"
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
      v-else-if="productsStore.trashedItems.length === 0"
      title="La papelera está vacía"
      message="Los productos que elimines aparecerán acá y podrás restaurarlos en cualquier momento."
    />

    <ul
      v-else
      class="trash-list"
    >
      <li
        v-for="product in productsStore.trashedItems"
        :key="product.id"
        class="trash-item"
      >
        <div class="trash-info">
          <p class="trash-name">
            {{ product.name }}
          </p>
          <p class="trash-meta">
            {{ formatCentsAsPesos(product.sale_price_cents) }}
          </p>
          <p class="trash-meta">
            Eliminado el {{ dateFormatter.format(new Date(product.deleted_at as string)) }}
            <template v-if="product.deleted_by_email">
              por {{ product.deleted_by_email }}
            </template>
          </p>
        </div>

        <div class="trash-actions">
          <button
            type="button"
            class="btn btn--primary"
            @click="handleRestore(product.id)"
          >
            Restaurar
          </button>
        </div>

        <div
          v-if="restoreErrors[product.id]"
          class="trash-error-banner"
        >
          <p
            class="banner banner--error"
            role="alert"
          >
            {{ restoreErrors[product.id] }}
          </p>
          <p class="trash-error-suggestion">
            Ajustá el inventario de este producto antes de restaurarlo.
          </p>
        </div>
      </li>
    </ul>
  </main>
</template>

<style scoped>
.trash-view {
  max-width: 960px;
  margin: 0 auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.trash-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.trash-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.trash-item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.trash-info {
  flex: 1;
  min-width: 12rem;
}

.trash-name {
  font-weight: 600;
}

.trash-meta {
  font-size: 0.875rem;
  color: var(--color-text-muted);
}

.trash-error-banner {
  flex-basis: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.trash-error-suggestion {
  font-size: 0.875rem;
  color: var(--color-text-muted);
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
  display: inline-flex;
  align-items: center;
}

.btn--primary {
  background: var(--color-primary);
  color: hsla(0, 0%, 100%, 1);
  border-color: transparent;
}
</style>
