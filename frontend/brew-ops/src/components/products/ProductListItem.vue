<script setup lang="ts">
// Presentational only — knows nothing about fetching/filtering. This
// separation is deliberate: a future view-mode switch (list/grid) only
// needs a new component like this one, never a change to ProductsView's
// data layer. See src/stores/products.ts for where products actually come
// from.
import { useI18n } from 'vue-i18n'
import type { Product } from '../../stores/products'
import StatusBadge from '../shared/StatusBadge.vue'
import { formatCentsAsPesos } from '../../utils/money'
import { PRODUCT_CATEGORY_I18N_KEYS } from '../../utils/productCategory'

const { t } = useI18n()

const props = defineProps<{
  product: Product
}>()

defineEmits<{
  edit: [product: Product]
  delete: [product: Product]
}>()

const isLowStock = props.product.current_stock <= props.product.min_stock
</script>

<template>
  <li class="product-row">
    <div class="product-info">
      <p class="product-name">
        {{ product.name }}
      </p>
      <p class="product-meta">
        {{ t(PRODUCT_CATEGORY_I18N_KEYS[product.category]) }} ·
        {{ formatCentsAsPesos(product.sale_price_cents) }}
      </p>
      <p class="product-stock">
        {{ t('products.stockLabel', { stock: product.current_stock }) }}
        <StatusBadge
          v-if="isLowStock"
          :label="t('products.lowStockBadge')"
          variant="warning"
        />
      </p>
    </div>

    <div class="product-actions">
      <button
        type="button"
        class="product-action"
        @click="$emit('edit', product)"
      >
        {{ t('common.edit') }}
      </button>
      <button
        type="button"
        class="product-action product-action--danger"
        @click="$emit('delete', product)"
      >
        {{ t('common.delete') }}
      </button>
    </div>

    <img
      v-if="product.image_url"
      :src="product.image_url"
      :alt="product.name"
      class="product-thumb"
      loading="lazy"
    >
    <div
      v-else
      class="product-thumb product-thumb--placeholder"
      aria-hidden="true"
    />
  </li>
</template>

<style scoped>
.product-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.product-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.product-name {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-meta,
.product-stock {
  font-size: 0.875rem;
  color: var(--color-text-muted);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.product-actions {
  display: flex;
  gap: var(--space-2);
  flex-shrink: 0;
}

.product-action {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  font-size: 0.875rem;
}

.product-action--danger {
  color: var(--color-error);
  border-color: hsla(4, 75%, 50%, 0.3);
}

/* Compact thumbnail on the right of the row, per CLAUDE.md's "many
   products at a glance" list layout — not a large card grid. */
.product-thumb {
  flex-shrink: 0;
  width: 3rem;
  height: 3rem;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
}

.product-thumb--placeholder {
  background: var(--color-surface-alt);
}
</style>
