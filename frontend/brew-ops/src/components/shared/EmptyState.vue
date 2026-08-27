<script setup lang="ts">
defineProps<{
  message: string
}>()
</script>

<template>
  <div class="empty-state">
    <p>{{ message }}</p>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-5);
  color: var(--color-text-muted);
  text-align: center;
}
</style>
