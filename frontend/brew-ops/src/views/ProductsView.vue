<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import EmptyState from '../components/shared/EmptyState.vue'
import SkeletonList from '../components/shared/SkeletonList.vue'
import ProductListItem from '../components/products/ProductListItem.vue'
import ProductFormModal from '../components/products/ProductFormModal.vue'
import { useProductsStore, type Product, type ProductCategory } from '../stores/products'
import { PRODUCT_CATEGORIES, PRODUCT_CATEGORY_I18N_KEYS } from '../utils/productCategory'

const { t } = useI18n()
const productsStore = useProductsStore()
const router = useRouter()

const isFiltered = computed(
  () => productsStore.categoryFilter !== null || productsStore.searchQuery.trim() !== '',
)

const showModal = ref(false)
const modalMode = ref<'create' | 'edit'>('create')
const editingProduct = ref<Product | null>(null)
const deleteError = ref('')
const deepLinkError = ref('')

onMounted(() => {
  void productsStore.fetchProducts(1)
  void openFromDeepLink()
})

// Supports the Dashboard's low-stock alerts, which link here as
// /products?edit=<id> to open the edit modal directly rather than making
// the owner find the product in the list themselves. Fetches the product
// by id independently of the paginated/filtered list — it may not be on
// whatever page happens to load first — then clears the query param so a
// page refresh or back-navigation doesn't reopen the modal unexpectedly.
async function openFromDeepLink() {
  const id = router.currentRoute.value.query.edit
  if (typeof id !== 'string' || id === '') {
    return
  }
  try {
    const product = await productsStore.getProduct(id)
    openEditModal(product)
  } catch {
    deepLinkError.value = t('products.deepLinkError')
  } finally {
    void router.replace({ path: '/products' })
  }
}

function openCreateModal() {
  modalMode.value = 'create'
  editingProduct.value = null
  showModal.value = true
}

function openEditModal(product: Product) {
  modalMode.value = 'edit'
  editingProduct.value = product
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingProduct.value = null
}

async function handleDelete(product: Product) {
  const confirmed = window.confirm(t('products.deleteConfirm', { name: product.name }))
  if (!confirmed) {
    return
  }
  deleteError.value = ''
  try {
    await productsStore.softDeleteProduct(product.id)
  } catch {
    deleteError.value = t('products.deleteError')
  }
}

function onCategoryFilterChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  productsStore.setCategoryFilter(value === '' ? null : (value as ProductCategory))
}

function retryLoad() {
  void productsStore.fetchProducts(productsStore.page)
}

function goToPage(page: number) {
  void productsStore.fetchProducts(page)
}
</script>

<template>
  <main class="products-view">
    <header class="products-header">
      <h1>{{ t('nav.products') }}</h1>
      <div class="products-header-actions">
        <RouterLink
          to="/products/trash"
          class="btn"
        >
          {{ t('common.viewTrash') }}
        </RouterLink>
        <button
          type="button"
          class="btn btn--primary"
          @click="openCreateModal"
        >
          {{ t('products.addProduct') }}
        </button>
      </div>
    </header>

    <div class="products-filters">
      <input
        :value="productsStore.searchQuery"
        type="search"
        :placeholder="t('products.searchPlaceholder')"
        :aria-label="t('products.searchAriaLabel')"
        @input="productsStore.searchQuery = ($event.target as HTMLInputElement).value"
      >
      <select
        :aria-label="t('products.filterAriaLabel')"
        @change="onCategoryFilterChange"
      >
        <option value="">
          {{ t('products.allCategories') }}
        </option>
        <option
          v-for="value in PRODUCT_CATEGORIES"
          :key="value"
          :value="value"
        >
          {{ t(PRODUCT_CATEGORY_I18N_KEYS[value]) }}
        </option>
      </select>
    </div>

    <p
      v-if="deleteError"
      class="banner banner--error"
      role="alert"
    >
      {{ deleteError }}
    </p>

    <p
      v-if="deepLinkError"
      class="banner banner--error"
      role="alert"
    >
      {{ deepLinkError }}
    </p>

    <SkeletonList
      v-if="productsStore.isLoading"
      :rows="6"
    />

    <EmptyState
      v-else-if="productsStore.loadError"
      :title="t('products.loadErrorTitle')"
      :message="t('common.genericErrorRetry')"
    >
      <template #action>
        <button
          type="button"
          class="btn"
          @click="retryLoad"
        >
          {{ t('common.retry') }}
        </button>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="productsStore.filteredItems.length === 0 && !isFiltered"
      :title="t('products.emptyTitle')"
      :message="t('products.emptyMessage')"
    >
      <template #action>
        <button
          type="button"
          class="btn btn--primary"
          @click="openCreateModal"
        >
          {{ t('products.addFirst') }}
        </button>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="productsStore.filteredItems.length === 0 && isFiltered"
      :title="t('products.noResultsTitle')"
      :message="t('products.noResultsMessage')"
    />

    <template v-else>
      <ul class="products-list">
        <ProductListItem
          v-for="product in productsStore.filteredItems"
          :key="product.id"
          :product="product"
          @edit="openEditModal"
          @delete="handleDelete"
        />
      </ul>

      <div class="pagination">
        <button
          type="button"
          class="btn"
          :disabled="productsStore.page <= 1"
          @click="goToPage(productsStore.page - 1)"
        >
          {{ t('products.prevPage') }}
        </button>
        <span>{{ t('products.pageLabel', { page: productsStore.page }) }}</span>
        <button
          type="button"
          class="btn"
          :disabled="!productsStore.hasMorePages"
          @click="goToPage(productsStore.page + 1)"
        >
          {{ t('products.nextPage') }}
        </button>
      </div>
    </template>

    <ProductFormModal
      v-if="showModal"
      :mode="modalMode"
      :product="editingProduct"
      @close="closeModal"
      @saved="closeModal"
    />
  </main>
</template>

<style scoped>
.products-view {
  max-width: 960px;
  margin: 0 auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.products-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.products-header-actions {
  display: flex;
  gap: var(--space-2);
}

.products-filters {
  display: flex;
  gap: var(--space-2);
}

.products-filters input,
.products-filters select {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.products-filters input {
  flex: 1;
}

.products-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
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

.btn:disabled {
  color: var(--color-text-muted);
  cursor: not-allowed;
}
</style>
