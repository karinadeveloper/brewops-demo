<script setup lang="ts">
// Generic status pill — used for the low-stock indicator now, and reusable
// later for sale sync states (PENDING_SYNC/SYNC_ERROR) without change.
defineProps<{
  label: string
  variant: 'warning' | 'error' | 'success' | 'neutral'
}>()
</script>

<template>
  <span
    class="status-badge"
    :class="`status-badge--${variant}`"
  >{{ label }}</span>
</template>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.125rem var(--space-2);
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
}

.status-badge--warning {
  color: hsla(40, 90%, 28%, 1);
  background: hsla(40, 90%, 50%, 0.18);
}

.status-badge--error {
  color: hsla(4, 75%, 30%, 1);
  background: hsla(4, 75%, 50%, 0.15);
}

.status-badge--success {
  color: hsla(145, 65%, 25%, 1);
  background: hsla(145, 65%, 38%, 0.15);
}

.status-badge--neutral {
  color: var(--color-text-muted);
  background: var(--color-surface-alt);
}
</style>
