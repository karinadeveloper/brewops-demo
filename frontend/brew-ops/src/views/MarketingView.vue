<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import EmptyState from '../components/shared/EmptyState.vue'
import SkeletonList from '../components/shared/SkeletonList.vue'
import MarketingAssetCard from '../components/marketing/MarketingAssetCard.vue'
import MarketingAssetFormModal from '../components/marketing/MarketingAssetFormModal.vue'
import { useMarketingStore, type MarketingAsset } from '../stores/marketing'

const { t } = useI18n()
const marketingStore = useMarketingStore()

const showModal = ref(false)
const deleteError = ref('')

onMounted(() => {
  void marketingStore.fetchAssets()
})

function retryLoad() {
  void marketingStore.fetchAssets()
}

async function handleDelete(asset: MarketingAsset) {
  const confirmed = window.confirm(t('marketing.deleteConfirm', { name: asset.name }))
  if (!confirmed) {
    return
  }
  deleteError.value = ''
  try {
    await marketingStore.softDeleteAsset(asset.id)
  } catch {
    deleteError.value = t('marketing.deleteError')
  }
}
</script>

<template>
  <main class="marketing-view">
    <header class="marketing-header">
      <h1>{{ t('nav.marketing') }}</h1>
      <div class="marketing-header-actions">
        <RouterLink
          to="/marketing/trash"
          class="btn"
        >
          {{ t('common.viewTrash') }}
        </RouterLink>
        <button
          type="button"
          class="btn btn--primary"
          @click="showModal = true"
        >
          {{ t('marketing.addImage') }}
        </button>
      </div>
    </header>

    <p
      v-if="deleteError"
      class="banner banner--error"
      role="alert"
    >
      {{ deleteError }}
    </p>

    <SkeletonList
      v-if="marketingStore.isLoading"
      :rows="3"
    />

    <EmptyState
      v-else-if="marketingStore.loadError"
      :title="t('marketing.loadErrorTitle')"
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
      v-else-if="marketingStore.items.length === 0"
      :title="t('marketing.emptyTitle')"
      :message="t('marketing.emptyMessage')"
    >
      <template #action>
        <button
          type="button"
          class="btn btn--primary"
          @click="showModal = true"
        >
          {{ t('marketing.addFirst') }}
        </button>
      </template>
    </EmptyState>

    <ul
      v-else
      class="asset-grid"
    >
      <MarketingAssetCard
        v-for="asset in marketingStore.items"
        :key="asset.id"
        :asset="asset"
        @delete="handleDelete"
      />
    </ul>

    <MarketingAssetFormModal
      v-if="showModal"
      @close="showModal = false"
      @saved="showModal = false"
    />
  </main>
</template>

<style scoped>
.marketing-view {
  max-width: 1100px;
  margin: 0 auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.marketing-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.marketing-header-actions {
  display: flex;
  gap: var(--space-2);
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
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
</style>
