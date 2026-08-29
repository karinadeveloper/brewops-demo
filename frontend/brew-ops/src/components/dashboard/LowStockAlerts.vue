<script setup lang="ts">
import { RouterLink } from 'vue-router'
import SkeletonList from '../shared/SkeletonList.vue'
import EmptyState from '../shared/EmptyState.vue'
import type { Product } from '../../stores/products'

defineProps<{
  items: Product[]
  isLoading: boolean
  error: boolean
}>()

defineEmits<{ retry: [] }>()
</script>

<template>
  <section
    class="low-stock"
    aria-label="Alertas de stock bajo"
  >
    <h2>Stock bajo</h2>

    <SkeletonList
      v-if="isLoading"
      :rows="3"
    />

    <EmptyState
      v-else-if="error"
      title="No se pudieron cargar las alertas"
      message="Ocurrió un error al conectar con el servidor."
    >
      <template #action>
        <button
          type="button"
          class="btn"
          @click="$emit('retry')"
        >
          Reintentar
        </button>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="items.length === 0"
      title="Todo el inventario está en buen nivel"
      message="Ningún producto está en o por debajo de su stock mínimo."
    />

    <ul
      v-else
      class="low-stock-list"
    >
      <li
        v-for="product in items"
        :key="product.id"
        class="low-stock-item"
      >
        <div class="low-stock-info">
          <span class="low-stock-name">{{ product.name }}</span>
          <span class="low-stock-detail">
            Stock: {{ product.current_stock }} · mínimo: {{ product.min_stock }}
          </span>
        </div>
        <RouterLink
          :to="`/products?edit=${product.id}`"
          class="btn"
        >
          Editar
        </RouterLink>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.low-stock {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.low-stock-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.low-stock-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.low-stock-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.low-stock-name {
  font-weight: 600;
}

.low-stock-detail {
  font-size: 0.85rem;
  color: var(--color-warning);
}

.btn {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
  white-space: nowrap;
}
</style>
