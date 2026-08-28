<script setup lang="ts">
import { onMounted, ref } from 'vue'
import EmptyState from '../components/shared/EmptyState.vue'
import ProductGridButton from '../components/pos/ProductGridButton.vue'
import CartPanel from '../components/pos/CartPanel.vue'
import { useProductsStore } from '../stores/products'
import { useSalesStore } from '../stores/sales'
import { useCart } from '../composables/useCart'
import { useConnectivity } from '../composables/useConnectivity'
import { addPendingSale } from '../composables/useOfflineSalesDb'
import { formatCentsAsPesos } from '../utils/money'
import type { ApiError } from '../composables/useApi'

const productsStore = useProductsStore()
const salesStore = useSalesStore()
const cart = useCart()
const { isOnline } = useConnectivity()

const paymentMethod = ref<'CASH' | 'TRANSFER'>('CASH')
const isCartOpen = ref(false)
const isSubmitting = ref(false)
const submitError = ref('')
const successMessage = ref('')
let successTimeoutId: ReturnType<typeof setTimeout> | null = null

onMounted(() => {
  void productsStore.fetchProducts(1)
})

function showSuccess(message: string) {
  successMessage.value = message
  if (successTimeoutId) {
    clearTimeout(successTimeoutId)
  }
  successTimeoutId = setTimeout(() => {
    successMessage.value = ''
  }, 4000)
}

// A thrown value shaped like ApiError (has a numeric status) means the
// backend actually responded — a genuine business rejection (e.g. 409
// insufficient stock). Anything else means fetch() itself failed before
// getting a response, i.e. we weren't really online despite what
// useConnectivity's last check said — safest to queue it offline rather
// than lose the sale.
function isBackendError(err: unknown): err is ApiError {
  return typeof err === 'object' && err !== null && typeof (err as ApiError).status === 'number'
}

async function confirmSale() {
  if (cart.isEmpty.value || isSubmitting.value) {
    return
  }

  const items = cart.items.value.map((item) => ({
    product_id: item.productId,
    quantity: item.quantity,
    unit_price_cents: item.unitPriceCents,
  }))

  // Generated once per confirmation, here — not at sync time — so it
  // survives a lost response and every retry of this same sale (online
  // fetch failure re-queued below, or a later "Reintentar") carries the
  // same key. The backend uses it to detect and no-op a duplicate attempt
  // instead of recording the sale twice. See CLAUDE.md's Business rules.
  const idempotencyKey = crypto.randomUUID()

  submitError.value = ''
  isSubmitting.value = true
  try {
    if (isOnline.value) {
      await salesStore.createSale({ items, payment_method: paymentMethod.value, idempotency_key: idempotencyKey })
      cart.clear()
      isCartOpen.value = false
      showSuccess('Venta registrada.')
    } else {
      await queueOffline(items, idempotencyKey)
    }
  } catch (err) {
    if (isBackendError(err)) {
      // A legitimate business rejection (e.g. insufficient stock) — the
      // user resolves it right here (adjust quantities, remove the item),
      // it's never treated as a sync/offline problem.
      submitError.value = err.message
    } else {
      // fetch() itself failed — we're not actually reachable. Fall back to
      // the offline path instead of losing the sale. This used to risk a
      // duplicate sale if the request had actually succeeded server-side
      // before the response was lost; now that the same idempotencyKey
      // travels with every retry (including the one useSaleSync makes once
      // connectivity returns), the backend recognizes the repeat and
      // returns the original sale instead of creating a second one.
      await queueOffline(items, idempotencyKey)
    }
  } finally {
    isSubmitting.value = false
  }
}

async function queueOffline(
  items: { product_id: string; quantity: number; unit_price_cents: number }[],
  idempotencyKey: string,
) {
  await addPendingSale({
    id: crypto.randomUUID(),
    items,
    payment_method: paymentMethod.value,
    idempotencyKey,
    createdAt: new Date().toISOString(),
    status: 'PENDING_SYNC',
  })
  cart.clear()
  isCartOpen.value = false
  showSuccess('Venta guardada. Se sincronizará cuando vuelva la conexión.')
}
</script>

<template>
  <main class="pos-view">
    <h1>Punto de venta</h1>

    <p
      v-if="successMessage"
      class="banner banner--success"
      role="status"
    >
      {{ successMessage }}
    </p>
    <p
      v-if="!isOnline"
      class="banner banner--warning"
      role="status"
    >
      Sin conexión — las ventas se guardarán y se sincronizarán automáticamente.
    </p>

    <EmptyState
      v-if="productsStore.hasLoadedOnce && productsStore.items.length === 0 && !productsStore.loadError"
      title="No hay productos disponibles"
      message="Agregá productos desde la sección de Productos antes de vender."
    />

    <div
      v-else
      class="pos-layout"
    >
      <section
        class="product-grid"
        aria-label="Productos"
      >
        <ProductGridButton
          v-for="product in productsStore.items"
          :key="product.id"
          :product="product"
          @add="cart.addProduct"
        />
      </section>

      <button
        v-if="!cart.isEmpty.value"
        type="button"
        class="cart-bubble"
        @click="isCartOpen = true"
      >
        {{ cart.itemCount.value }} · {{ formatCentsAsPesos(cart.totalCents.value) }}
      </button>

      <CartPanel
        :items="cart.items.value"
        :total-cents="cart.totalCents.value"
        :payment-method="paymentMethod"
        :is-open="isCartOpen"
        :is-submitting="isSubmitting"
        :submit-error="submitError"
        @update:payment-method="paymentMethod = $event"
        @remove="cart.removeItem"
        @set-quantity="cart.setQuantity"
        @confirm="confirmSale"
        @close="isCartOpen = false"
      />
    </div>
  </main>
</template>

<style scoped>
.pos-view {
  max-width: 1100px;
  margin: 0 auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.pos-layout {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.product-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(7rem, 1fr));
  gap: var(--space-2);
}

.cart-bubble {
  position: fixed;
  right: var(--space-3);
  bottom: var(--space-3);
  padding: var(--space-2) var(--space-4);
  border-radius: 999px;
  background: var(--color-primary);
  color: hsla(0, 0%, 100%, 1);
  font-weight: 600;
  box-shadow: 0 4px 12px hsla(220, 15%, 15%, 0.2);
  z-index: 15;
}

.banner {
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: 0.9rem;
}

.banner--success {
  color: hsla(145, 65%, 25%, 1);
  background: hsla(145, 65%, 38%, 0.15);
}

.banner--warning {
  color: hsla(40, 90%, 28%, 1);
  background: hsla(40, 90%, 50%, 0.18);
}

@media (min-width: 768px) {
  .pos-layout {
    flex-direction: row;
    align-items: flex-start;
  }

  .cart-bubble {
    display: none;
  }
}
</style>
