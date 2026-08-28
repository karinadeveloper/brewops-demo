import { computed, ref } from 'vue'
import type { Product } from '../stores/products'

export interface CartItem {
  productId: string
  name: string
  unitPriceCents: number
  quantity: number
  imageUrl: string | null
}

// A fresh cart per call — unlike useConnectivity/useSaleSync, this is
// scoped to whichever view calls it (PointOfSaleView), not an app-wide
// singleton. The caller shares the single returned instance with its
// child components via props, so the grid, the sidebar panel, and the
// mobile bottom sheet all read/mutate the same cart.
export function useCart() {
  const items = ref<CartItem[]>([])

  function addProduct(product: Product) {
    const existing = items.value.find((item) => item.productId === product.id)
    if (existing) {
      existing.quantity += 1
      return
    }
    items.value.push({
      productId: product.id,
      name: product.name,
      unitPriceCents: product.sale_price_cents,
      quantity: 1,
      imageUrl: product.image_url,
    })
  }

  function removeItem(productId: string) {
    items.value = items.value.filter((item) => item.productId !== productId)
  }

  function setQuantity(productId: string, quantity: number) {
    if (quantity <= 0) {
      removeItem(productId)
      return
    }
    const item = items.value.find((i) => i.productId === productId)
    if (item) {
      item.quantity = quantity
    }
  }

  function clear() {
    items.value = []
  }

  const totalCents = computed(() =>
    items.value.reduce((sum, item) => sum + item.unitPriceCents * item.quantity, 0),
  )
  const itemCount = computed(() => items.value.reduce((sum, item) => sum + item.quantity, 0))
  const isEmpty = computed(() => items.value.length === 0)

  return { items, addProduct, removeItem, setQuantity, clear, totalCents, itemCount, isEmpty }
}
