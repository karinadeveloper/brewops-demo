<script setup lang="ts">
// Mirrors ProductFormModal's upload UX (preview on selection, save
// disabled while the request is in flight, a clear Content-Type error) —
// see stores/marketing.ts's createAsset for why the actual network shape
// differs (one combined multipart POST here, vs. product's separate
// pre-upload-then-create flow): there is no generic "upload first, attach
// the URL later" endpoint for marketing assets.
import { computed, ref } from 'vue'
import type { ApiError } from '../../composables/useApi'
import { useMarketingStore, type MarketingAsset } from '../../stores/marketing'
import { MARKETING_ASSET_TYPES, MARKETING_ASSET_TYPE_LABELS } from '../../utils/marketingAssetType'

const emit = defineEmits<{
  close: []
  saved: [asset: MarketingAsset]
}>()

const marketingStore = useMarketingStore()

const name = ref('')
const type = ref<'PRODUCT' | 'PROMOTION'>('PRODUCT')
const selectedFile = ref<File | null>(null)
const previewUrl = ref<string | null>(null)

const errors = ref<{ name?: string; file?: string }>({})
const isSaving = ref(false)
const saveError = ref('')

function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) {
    return
  }
  selectedFile.value = file
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
  }
  previewUrl.value = URL.createObjectURL(file)
  errors.value.file = undefined
}

const canSave = computed(() => !isSaving.value)

function validate(): boolean {
  const nextErrors: { name?: string; file?: string } = {}
  if (name.value.trim() === '') {
    nextErrors.name = 'Ingresá un nombre.'
  }
  if (!selectedFile.value) {
    nextErrors.file = 'Seleccioná una imagen.'
  }
  errors.value = nextErrors
  return Object.keys(nextErrors).length === 0
}

async function handleSubmit() {
  if (!validate() || !selectedFile.value) {
    return
  }
  isSaving.value = true
  saveError.value = ''
  try {
    const asset = await marketingStore.createAsset(name.value.trim(), type.value, selectedFile.value)
    emit('saved', asset)
    emit('close')
  } catch (err) {
    const apiError = err as ApiError
    saveError.value =
      apiError.status === 400
        ? 'La imagen debe ser un archivo JPEG, PNG o WebP.'
        : 'No se pudo subir la imagen. Intentá de nuevo.'
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
      <h2>Subir imagen</h2>

      <form
        class="asset-form"
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
          <span>Nombre</span>
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
          <span>Tipo</span>
          <select
            v-model="type"
            :disabled="isSaving"
          >
            <option
              v-for="value in MARKETING_ASSET_TYPES"
              :key="value"
              :value="value"
            >
              {{ MARKETING_ASSET_TYPE_LABELS[value] }}
            </option>
          </select>
        </label>

        <label class="field">
          <span>Imagen</span>
          <input
            type="file"
            accept="image/jpeg,image/png,image/webp"
            :disabled="isSaving"
            @change="onFileSelected"
          >
          <small
            v-if="errors.file"
            class="field-error"
          >{{ errors.file }}</small>
        </label>
        <img
          v-if="previewUrl"
          :src="previewUrl"
          alt="Vista previa"
          class="image-preview"
        >

        <div class="modal-actions">
          <button
            type="button"
            class="btn"
            :disabled="isSaving"
            @click="$emit('close')"
          >
            Cancelar
          </button>
          <button
            type="submit"
            class="btn btn--primary"
            :disabled="!canSave"
          >
            {{ isSaving ? 'Subiendo…' : 'Subir imagen' }}
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

.asset-form {
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
</style>
