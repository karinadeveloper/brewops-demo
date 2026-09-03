<script setup lang="ts">
// Presentational cart. The same markup serves both layouts: a
// permanently-visible sidebar on tablet/laptop and a bottom sheet on
// narrow screens, switched purely by CSS media queries (see the scoped
// style) — isOpen only matters for the mobile transform/close button;
// desktop widths force the panel visible regardless of it.
import { useI18n } from 'vue-i18n'
import type { CartItem } from '../../composables/useCart'
import { formatCentsAsPesos } from '../../utils/money'

const { t } = useI18n()

defineProps<{
  items: CartItem[]
  totalCents: number
  paymentMethod: 'CASH' | 'TRANSFER'
  isOpen: boolean
  isSubmitting: boolean
  submitError: string
}>()

const emit = defineEmits<{
  'update:paymentMethod': [value: 'CASH' | 'TRANSFER']
  remove: [productId: string]
  setQuantity: [productId: string, quantity: number]
  confirm: []
  close: []
}>()

function onQuantityInput(productId: string, event: Event) {
  const value = Number((event.target as HTMLInputElement).value)
  emit('setQuantity', productId, value)
}
</script>

<template>
  <aside
    class="cart-panel"
    :class="{ 'cart-panel--open': isOpen }"
    :aria-label="t('cart.ariaLabel')"
  >
    <div class="cart-header">
      <h2>{{ t('cart.title') }}</h2>
      <button
        type="button"
        class="cart-close"
        @click="$emit('close')"
      >
        {{ t('common.close') }}
      </button>
    </div>

    <p
      v-if="items.length === 0"
      class="cart-empty"
    >
      {{ t('cart.empty') }}
    </p>

    <ul
      v-else
      class="cart-items"
    >
      <li
        v-for="item in items"
        :key="item.productId"
        class="cart-item"
      >
        <span class="cart-item-name">{{ item.name }}</span>
        <input
          type="number"
          min="1"
          class="cart-item-quantity"
          :value="item.quantity"
          :aria-label="t('pos.quantityAriaLabel', { name: item.name })"
          @change="onQuantityInput(item.productId, $event)"
        >
        <span class="cart-item-subtotal">{{ formatCentsAsPesos(item.unitPriceCents * item.quantity) }}</span>
        <button
          type="button"
          class="cart-item-remove"
          :aria-label="t('pos.removeAriaLabel', { name: item.name })"
          @click="$emit('remove', item.productId)"
        >
          ✕
        </button>
      </li>
    </ul>

    <div class="cart-total">
      <span>{{ t('cart.total') }}</span>
      <strong>{{ formatCentsAsPesos(totalCents) }}</strong>
    </div>

    <fieldset class="payment-method">
      <legend>{{ t('cart.paymentMethodLegend') }}</legend>
      <label>
        <input
          type="radio"
          name="payment-method"
          value="CASH"
          :checked="paymentMethod === 'CASH'"
          @change="$emit('update:paymentMethod', 'CASH')"
        >
        {{ t('cart.cash') }}
      </label>
      <label>
        <input
          type="radio"
          name="payment-method"
          value="TRANSFER"
          :checked="paymentMethod === 'TRANSFER'"
          @change="$emit('update:paymentMethod', 'TRANSFER')"
        >
        {{ t('cart.transfer') }}
      </label>
    </fieldset>

    <p
      v-if="submitError"
      class="banner banner--error"
      role="alert"
    >
      {{ submitError }}
    </p>

    <button
      type="button"
      class="btn btn--primary confirm-button"
      :disabled="items.length === 0 || isSubmitting"
      @click="$emit('confirm')"
    >
      {{ isSubmitting ? t('cart.confirming') : t('cart.confirmSale') }}
    </button>
  </aside>
</template>

<style scoped>
.cart-panel {
  position: fixed;
  inset: auto 0 0 0;
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
  border-radius: var(--radius-md) var(--radius-md) 0 0;
  padding: var(--space-3);
  max-height: 80vh;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  transform: translateY(100%);
  transition: transform 0.2s ease;
  z-index: 20;
}

.cart-panel--open {
  transform: translateY(0);
}

.cart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.cart-empty {
  color: var(--color-text-muted);
}

.cart-items {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.cart-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.cart-item-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cart-item-quantity {
  width: 3.5rem;
  padding: var(--space-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.cart-item-subtotal {
  min-width: 4.5rem;
  text-align: right;
}

.cart-item-remove {
  color: var(--color-error);
}

.cart-total {
  display: flex;
  justify-content: space-between;
  font-size: 1.125rem;
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

.payment-method {
  display: flex;
  gap: var(--space-3);
  border: none;
  padding: 0;
}

.payment-method legend {
  width: 100%;
  font-size: 0.875rem;
  color: var(--color-text-muted);
  margin-bottom: var(--space-1);
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

.btn--primary {
  background: var(--color-primary);
  color: hsla(0, 0%, 100%, 1);
  border-color: transparent;
}

.btn--primary:disabled {
  background: var(--color-text-muted);
  cursor: not-allowed;
}

.confirm-button {
  width: 100%;
}

/* Mobile default: cart-close is meaningful (bottom sheet). At tablet/
   desktop widths the panel becomes a permanently-visible sidebar — see
   PointOfSaleView's .pos-layout for the flex context this sits in. */
@media (min-width: 768px) {
  .cart-panel {
    position: sticky;
    top: 0;
    transform: none !important;
    width: 20rem;
    max-height: none;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
    flex-shrink: 0;
  }

  .cart-close {
    display: none;
  }
}
</style>
