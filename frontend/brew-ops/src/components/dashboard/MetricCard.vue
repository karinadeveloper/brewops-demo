<script setup lang="ts">
defineProps<{
  title: string
  value: string
  isLoading: boolean
  error: boolean
}>()

defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="metric-card">
    <h3 class="metric-title">
      {{ title }}
    </h3>
    <div
      v-if="isLoading"
      class="metric-skeleton"
      aria-hidden="true"
    />
    <p
      v-else-if="error"
      class="metric-error"
    >
      No se pudo cargar.
      <button
        type="button"
        class="btn-link"
        @click="$emit('retry')"
      >
        Reintentar
      </button>
    </p>
    <p
      v-else
      class="metric-value"
    >
      {{ value }}
    </p>
  </div>
</template>

<style scoped>
.metric-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.metric-title {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.02em;
}

.metric-value {
  font-size: 1.75rem;
  font-weight: 700;
}

.metric-error {
  font-size: 0.9rem;
  color: var(--color-error);
}

.metric-skeleton {
  height: 1.75rem;
  width: 60%;
  border-radius: var(--radius-sm);
  background: var(--color-surface-alt);
  animation: metric-pulse 1.4s ease-in-out infinite;
}

.btn-link {
  border: none;
  background: none;
  color: var(--color-primary);
  padding: 0;
  font-size: 0.85rem;
  text-decoration: underline;
}

@keyframes metric-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
