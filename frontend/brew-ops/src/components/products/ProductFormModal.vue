<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useApi, type ApiError } from '../../composables/useApi'
import { useProductsStore, type Product, type ProductCategory, type ProductInput } from '../../stores/products'
import { centsToPesos, pesosToCents } from '../../utils/money'
import { PRODUCT_CATEGORIES, PRODUCT_CATEGORY_I18N_KEYS } from '../../utils/productCategory'
import {
  hasFormErrors,
  validateProductForm,
  type ProductFormErrors,
  type ProductFormValues,
} from '../../utils/productFormValidation'

const props = defineProps<{
  mode: 'create' | 'edit'
  product?: Product | null
}>()

const emit = defineEmits<{
  close: []
  saved: [product: Product]
}>()

const { t } = useI18n()
const productsStore = useProductsStore()

const name = ref(props.product?.name ?? '')
const category = ref<ProductCategory>(props.product?.category ?? 'juice')
const salePricePesos = ref<number | null>(
  props.product ? centsToPesos(props.product.sale_price_cents) : null,
)
const costPesos = ref<number | null>(props.product ? centsToPesos(props.product.cost_cents) : null)
const currentStock = ref<number | null>(props.product?.current_stock ?? null)
const minStock = ref<number | null>(props.product?.min_stock ?? null)

const errors = ref<ProductFormErrors>({})
const isSaving = ref(false)
const saveError = ref('')
const conflictMessage = ref('')

// --- Image upload — uploads immediately on selection, independent of the
// main form submit, so the preview and any upload error are visible right
// away rather than only after clicking "Guardar". ---
const imageUrl = ref<string | null>(props.product?.image_url ?? null)
const previewUrl = ref<string | null>(props.product?.image_url ?? null)
const selectedFile = ref<File | null>(null)
const isUploadingImage = ref(false)
const uploadError = ref('')

function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) {
    return
  }
  selectedFile.value = file
  if (previewUrl.value?.startsWith('blob:')) {
    URL.revokeObjectURL(previewUrl.value)
  }
  previewUrl.value = URL.createObjectURL(file)
  void uploadImage(file)
}

async function uploadImage(file: File) {
  isUploadingImage.value = true
  uploadError.value = ''
  try {
    const formData = new FormData()
    formData.append('image', file)
    const { postForm } = useApi()
    const response = await postForm<{ url: string }>('/uploads/image', formData)
    imageUrl.value = response.url
  } catch {
    uploadError.value = t('productForm.uploadError')
  } finally {
    isUploadingImage.value = false
  }
}

function retryUpload() {
  if (selectedFile.value) {
    void uploadImage(selectedFile.value)
  }
}

// canSave is false while the image upload is still in flight, so a product
// is never created/updated pointing at an image_url that doesn't exist yet.
const canSave = computed(() => !isUploadingImage.value && !isSaving.value)

async function handleSubmit() {
  const values: ProductFormValues = {
    name: name.value,
    category: category.value,
    salePricePesos: salePricePesos.value,
    costPesos: costPesos.value,
    currentStock: currentStock.value,
    minStock: minStock.value,
  }
  errors.value = validateProductForm(values)
  if (hasFormErrors(errors.value)) {
    return
  }

  isSaving.value = true
  saveError.value = ''
  try {
    const input: ProductInput = {
      name: values.name.trim(),
      category: values.category,
      sale_price_cents: pesosToCents(values.salePricePesos as number),
      cost_cents: pesosToCents(values.costPesos as number),
      current_stock: values.currentStock as number,
      min_stock: values.minStock as number,
      image_url: imageUrl.value,
    }

    const saved =
      props.mode === 'edit' && props.product
        ? await productsStore.updateProduct(props.product.id, props.product.version, input)
        : await productsStore.createProduct(input)

    emit('saved', saved)
    emit('close')
  } catch (err) {
    const apiError = err as ApiError
    if (apiError.status === 409) {
      // Per CLAUDE.md: don't leave the user retrying blindly against a
      // conflict that won't resolve itself. Refresh the underlying list in
      // the background and replace the Save action with just a Close
      // button — see the template below.
      conflictMessage.value = t('productForm.conflictMessage')
      void productsStore.fetchProducts(productsStore.page)
    } else {
      saveError.value = t('productForm.saveError')
    }
  } finally {
    isSaving.value = false
  }
}
</script>

<template>
  <div
    class="modal-backdrop"
    role="presentation"
    @click.self="$emit('close')"
  >
    <div
      class="modal"
      role="dialog"
      aria-modal="true"
    >
      <h2>{{ mode === 'edit' ? t('productForm.titleEdit') : t('products.addProduct') }}</h2>

      <template v-if="conflictMessage">
        <p
          class="banner banner--error"
          role="alert"
        >
          {{ conflictMessage }}
        </p>
        <div class="modal-actions">
          <button
            type="button"
            class="btn"
            @click="$emit('close')"
          >
            {{ t('common.close') }}
          </button>
        </div>
      </template>

      <form
        v-else
        class="product-form"
        @submit.prevent="handleSubmit"
      >
        <p
          v-if="saveError"
          class="banner banner--error"
          role="alert"
        >
          {{ saveError }}
        </p>

        <label class="field">
          <span>{{ t('common.name') }}</span>
          <input
            v-model="name"
            type="text"
            :disabled="isSaving"
          >
          <small
            v-if="errors.name"
            class="field-error"
          >{{ errors.name }}</small>
        </label>

        <label class="field">
          <span>{{ t('productForm.category') }}</span>
          <select
            v-model="category"
            :disabled="isSaving"
          >
            <option
              v-for="value in PRODUCT_CATEGORIES"
              :key="value"
              :value="value"
            >
              {{ t(PRODUCT_CATEGORY_I18N_KEYS[value]) }}
            </option>
          </select>
        </label>

        <label class="field">
          <span>{{ t('productForm.salePrice') }}</span>
          <input
            v-model.number="salePricePesos"
            type="number"
            step="0.01"
            min="0"
            :disabled="isSaving"
          >
          <small
            v-if="errors.salePricePesos"
            class="field-error"
          >{{ errors.salePricePesos }}</small>
        </label>

        <label class="field">
          <span>{{ t('productForm.cost') }}</span>
          <input
            v-model.number="costPesos"
            type="number"
            step="0.01"
            min="0"
            :disabled="isSaving"
          >
          <small
            v-if="errors.costPesos"
            class="field-error"
          >{{ errors.costPesos }}</small>
        </label>

        <label class="field">
          <span>{{ t('productForm.currentStock') }}</span>
          <input
            v-model.number="currentStock"
            type="number"
            step="1"
            min="0"
            :disabled="isSaving"
          >
          <small
            v-if="errors.currentStock"
            class="field-error"
          >{{ errors.currentStock }}</small>
        </label>

        <label class="field">
          <span>{{ t('productForm.minStock') }}</span>
          <input
            v-model.number="minStock"
            type="number"
            step="1"
            min="0"
            :disabled="isSaving"
          >
          <small
            v-if="errors.minStock"
            class="field-error"
          >{{ errors.minStock }}</small>
        </label>

        <label class="field">
          <span>{{ t('common.image') }}</span>
          <input
            type="file"
            accept="image/jpeg,image/png,image/webp"
            :disabled="isSaving"
            @change="onFileSelected"
          >
        </label>
        <img
          v-if="previewUrl"
          :src="previewUrl"
          :alt="t('common.imagePreviewAlt')"
          class="image-preview"
        >
        <p
          v-if="isUploadingImage"
          class="upload-status"
        >
          {{ t('productForm.uploading') }}
        </p>
        <template v-if="uploadError">
          <p
            class="field-error"
            role="alert"
          >
            {{ uploadError }}
          </p>
          <button
            type="button"
            class="btn btn--link"
            @click="retryUpload"
          >
            {{ t('productForm.retryUpload') }}
          </button>
        </template>

        <div class="modal-actions">
          <button
            type="button"
            class="btn"
            :disabled="isSaving"
            @click="$emit('close')"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            class="btn btn--primary"
            :disabled="!canSave"
          >
            {{ isSaving ? t('productForm.saving') : mode === 'edit' ? t('productForm.saveChanges') : t('productForm.createProduct') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: hsla(220, 15%, 15%, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-3);
  z-index: 10;
}

.modal {
  width: 100%;
  max-width: 28rem;
  max-height: 90vh;
  overflow-y: auto;
  background: var(--color-surface);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.product-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: 0.9rem;
}

.field input,
.field select {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.field-error {
  color: var(--color-error);
}

.image-preview {
  width: 6rem;
  height: 6rem;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
}

.upload-status {
  color: var(--color-text-muted);
  font-size: 0.875rem;
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

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
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

.btn--link {
  border: none;
  color: var(--color-primary);
  padding: 0;
  align-self: flex-start;
}
</style>
