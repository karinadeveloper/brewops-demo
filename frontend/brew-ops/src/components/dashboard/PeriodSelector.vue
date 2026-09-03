<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { PERIOD_VALUES, PERIOD_I18N_KEYS, type Period } from '../../utils/period'

const { t } = useI18n()

defineProps<{ modelValue: Period }>()
const emit = defineEmits<{ 'update:modelValue': [value: Period] }>()
</script>

<template>
  <div
    class="period-selector"
    role="group"
    :aria-label="t('period.selectorLabel')"
  >
    <button
      v-for="value in PERIOD_VALUES"
      :key="value"
      type="button"
      class="period-btn"
      :class="{ 'period-btn--active': modelValue === value }"
      :aria-pressed="modelValue === value"
      @click="emit('update:modelValue', value)"
    >
      {{ t(PERIOD_I18N_KEYS[value]) }}
    </button>
  </div>
</template>

<style scoped>
.period-selector {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
}

.period-btn {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 0.875rem;
}

.period-btn--active {
  background: var(--color-primary);
  border-color: transparent;
  color: hsla(0, 0%, 100%, 1);
  font-weight: 600;
}
</style>
