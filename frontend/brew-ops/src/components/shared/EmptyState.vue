<script setup lang="ts">
// title/message-only usage (e.g. DashboardView's placeholder) still works
// unchanged — the action slot is purely additive, for views that need a
// suggested next step (e.g. "Agregar tu primer producto").
defineProps<{
  title?: string
  message: string
}>()
</script>

<template>
  <div class="empty-state">
    <h3
      v-if="title"
      class="empty-state-title"
    >
      {{ title }}
    </h3>
    <p>{{ message }}</p>
    <div
      v-if="$slots.action"
      class="empty-state-action"
    >
      <slot name="action" />
    </div>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-5);
  color: var(--color-text-muted);
  text-align: center;
}

.empty-state-title {
  color: var(--color-text);
}

.empty-state-action {
  margin-top: var(--space-2);
}
</style>
