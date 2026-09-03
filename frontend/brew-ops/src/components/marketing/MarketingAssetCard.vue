<script setup lang="ts">
// Presentational only, mirroring ProductListItem's split between data and
// display — MarketingView owns fetching/state, this only renders one card.
import { useI18n } from 'vue-i18n'
import type { MarketingAsset } from '../../stores/marketing'
import StatusBadge from '../shared/StatusBadge.vue'
import { MARKETING_ASSET_TYPE_I18N_KEYS } from '../../utils/marketingAssetType'

const { t } = useI18n()

defineProps<{ asset: MarketingAsset }>()
defineEmits<{ delete: [asset: MarketingAsset] }>()

const dateFormatter = new Intl.DateTimeFormat('es-MX', {
  dateStyle: 'medium',
  timeZone: 'America/Mexico_City',
})
</script>

<template>
  <li class="asset-card">
    <img
      :src="asset.image_url"
      :alt="asset.name"
      class="asset-image"
      loading="lazy"
    >
    <div class="asset-info">
      <p class="asset-name">
        {{ asset.name }}
      </p>
      <div class="asset-meta">
        <StatusBadge
          :label="t(MARKETING_ASSET_TYPE_I18N_KEYS[asset.type])"
          :variant="asset.type === 'PROMOTION' ? 'warning' : 'neutral'"
        />
        <span class="asset-date">{{ dateFormatter.format(new Date(asset.created_at)) }}</span>
      </div>
    </div>
    <button
      type="button"
      class="asset-delete"
      @click="$emit('delete', asset)"
    >
      {{ t('common.delete') }}
    </button>
  </li>
</template>

<style scoped>
.asset-card {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.asset-image {
  width: 100%;
  aspect-ratio: 4 / 3;
  object-fit: cover;
  background: var(--color-surface-alt);
}

.asset-info {
  padding: var(--space-2) var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  flex: 1;
}

.asset-name {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  font-size: 0.8rem;
  color: var(--color-text-muted);
}

.asset-delete {
  margin: 0 var(--space-3) var(--space-3);
  padding: var(--space-1) var(--space-2);
  border: 1px solid hsla(4, 75%, 50%, 0.3);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-error);
  font-size: 0.875rem;
  align-self: flex-start;
}
</style>
