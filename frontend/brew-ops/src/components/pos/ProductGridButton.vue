<script setup lang="ts">
// Presentational — a big touch target styled like a physical register key.
// Tapping always means "add one unit"; adjusting quantity for a larger
// order happens in the cart panel's own quantity input, not by repeated
// tapping here.
import type { Product } from '../../stores/products'
import { formatCentsAsPesos } from '../../utils/money'

defineProps<{
  product: Product
}>()

defineEmits<{
  add: [product: Product]
}>()
</script>

<template>
  <button
    type="button"
    class="product-key"
    @click="$emit('add', product)"
  >
    <img
      v-if="product.image_url"
      :src="product.image_url"
      :alt="product.name"
      class="product-key-image"
    >
    <div
      v-else
      class="product-key-image product-key-image--placeholder"
      aria-hidden="true"
    />
    <span class="product-key-name">{{ product.name }}</span>
    <span class="product-key-price">{{ formatCentsAsPesos(product.sale_price_cents) }}</span>
  </button>
</template>

<style scoped>
.product-key {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-2);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-align: center;
  /* Large touch target, per CLAUDE.md's cash-register-style grid. */
  min-height: 6.5rem;
}

.product-key:active {
  background: var(--color-surface-alt);
}

.product-key-image {
  width: 3rem;
  height: 3rem;
  object-fit: cover;
  border-radius: var(--radius-sm);
}

.product-key-image--placeholder {
  background: var(--color-surface-alt);
}

.product-key-name {
  font-weight: 600;
  font-size: 0.875rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.product-key-price {
  font-size: 0.8125rem;
  color: var(--color-text-muted);
}
</style>
