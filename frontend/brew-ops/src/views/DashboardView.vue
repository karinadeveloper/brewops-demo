<script setup lang="ts">
import EmptyState from '../components/shared/EmptyState.vue'
</script>

<template>
  <main class="dashboard">
    <h1>BrewOps</h1>
    <EmptyState message="El panel de control estará disponible próximamente." />
  </main>
</template>

<style scoped>
.dashboard {
  max-width: 960px;
  margin: 0 auto;
  padding: var(--space-4);
}
</style>
