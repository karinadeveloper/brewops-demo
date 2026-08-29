<script setup lang="ts">
import { PERIOD_OPTIONS, type Period } from '../../utils/period'

defineProps<{ modelValue: Period }>()
const emit = defineEmits<{ 'update:modelValue': [value: Period] }>()
</script>

<template>
  <div
    class="period-selector"
    role="group"
    aria-label="Seleccionar período"
  >
    <button
      v-for="option in PERIOD_OPTIONS"
      :key="option.value"
      type="button"
      class="period-btn"
      :class="{ 'period-btn--active': modelValue === option.value }"
      :aria-pressed="modelValue === option.value"
      @click="emit('update:modelValue', option.value)"
    >
      {{ option.label }}
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
