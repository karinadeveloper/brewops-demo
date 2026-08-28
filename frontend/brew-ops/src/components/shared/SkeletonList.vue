<script setup lang="ts">
// Generic list-loading placeholder — reusable for any row-based list
// (products now, sales/inventory movements later), not product-specific.
withDefaults(defineProps<{ rows?: number }>(), { rows: 6 })
</script>

<template>
  <ul
    class="skeleton-list"
    aria-hidden="true"
  >
    <li
      v-for="row in rows"
      :key="row"
      class="skeleton-row"
    >
      <div class="skeleton-thumb" />
      <div class="skeleton-lines">
        <div class="skeleton-line skeleton-line--wide" />
        <div class="skeleton-line skeleton-line--narrow" />
      </div>
    </li>
  </ul>
</template>

<style scoped>
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.skeleton-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.skeleton-thumb {
  flex-shrink: 0;
  width: 3rem;
  height: 3rem;
  border-radius: var(--radius-sm);
  background: var(--color-surface-alt);
  animation: skeleton-pulse 1.4s ease-in-out infinite;
}

.skeleton-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.skeleton-line {
  height: 0.75rem;
  border-radius: var(--radius-sm);
  background: var(--color-surface-alt);
  animation: skeleton-pulse 1.4s ease-in-out infinite;
}

.skeleton-line--wide {
  width: 60%;
}

.skeleton-line--narrow {
  width: 30%;
}

@keyframes skeleton-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
