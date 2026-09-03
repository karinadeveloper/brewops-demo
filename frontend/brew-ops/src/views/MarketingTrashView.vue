<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import EmptyState from '../components/shared/EmptyState.vue'
import SkeletonList from '../components/shared/SkeletonList.vue'
import { useMarketingStore } from '../stores/marketing'
import { MARKETING_ASSET_TYPE_I18N_KEYS } from '../utils/marketingAssetType'

const { t } = useI18n()
const marketingStore = useMarketingStore()

// Keyed by asset id, same as ProductsTrashView's restoreErrors — kept even
// though a marketing asset restore can't actually be rejected (no stock
// invariant to revalidate, unlike Product.restore), so a transport failure
// on one row still only shows an error for that row.
const restoreErrors = reactive<Record<string, string>>({})

const dateFormatter = new Intl.DateTimeFormat('es-MX', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'America/Mexico_City',
})

onMounted(() => {
  void marketingStore.fetchTrash()
})

async function handleRestore(id: string) {
  delete restoreErrors[id]
  try {
    await marketingStore.restoreAsset(id)
  } catch {
    restoreErrors[id] = t('marketingTrash.restoreError')
  }
}

function retryLoad() {
  void marketingStore.fetchTrash()
}
</script>

<template>
  <main class="trash-view">
    <header class="trash-header">
      <h1>{{ t('marketingTrash.title') }}</h1>
      <RouterLink
        to="/marketing"
        class="btn"
      >
        {{ t('marketingTrash.backToMarketing') }}
      </RouterLink>
    </header>

    <SkeletonList
      v-if="marketingStore.isLoadingTrash"
      :rows="4"
    />

    <EmptyState
      v-else-if="marketingStore.trashLoadError"
      :title="t('common.trashLoadError')"
      :message="t('common.genericError')"
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
      v-else-if="marketingStore.trashedItems.length === 0"
      :title="t('common.trashEmptyTitle')"
      :message="t('marketingTrash.emptyMessage')"
    />

    <ul
      v-else
      class="trash-list"
    >
      <li
        v-for="asset in marketingStore.trashedItems"
        :key="asset.id"
        class="trash-item"
      >
        <img
          :src="asset.image_url"
          :alt="asset.name"
          class="trash-thumb"
          loading="lazy"
        >

        <div class="trash-info">
          <p class="trash-name">
            {{ asset.name }}
          </p>
          <p class="trash-meta">
            {{ t(MARKETING_ASSET_TYPE_I18N_KEYS[asset.type]) }}
          </p>
          <p class="trash-meta">
            {{ t('marketingTrash.deletedOn', { date: dateFormatter.format(new Date(asset.deleted_at as string)) }) }}
            <template v-if="asset.deleted_by_email">
              {{ t('common.deletedBy', { email: asset.deleted_by_email }) }}
            </template>
          </p>
        </div>

        <div class="trash-actions">
          <button
            type="button"
            class="btn btn--primary"
            @click="handleRestore(asset.id)"
          >
            {{ t('common.restore') }}
          </button>
        </div>

        <p
          v-if="restoreErrors[asset.id]"
          class="banner banner--error trash-error-banner"
          role="alert"
        >
          {{ restoreErrors[asset.id] }}
        </p>
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

.trash-thumb {
  flex-shrink: 0;
  width: 3rem;
  height: 3rem;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
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
