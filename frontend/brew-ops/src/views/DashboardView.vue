<script setup lang="ts">
import { RouterLink } from 'vue-router'
import EmptyState from '../components/shared/EmptyState.vue'
</script>

<template>
  <main class="dashboard">
    <h1>BrewOps</h1>
    <!-- Minimal nav until Session 8 builds a real dashboard/shell — without
         this, /products is only reachable by typing the URL. -->
    <nav class="dashboard-nav">
      <RouterLink to="/products">
        Productos
      </RouterLink>
      <RouterLink to="/pos">
        Punto de venta
      </RouterLink>
      <RouterLink to="/sales">
        Historial de ventas
      </RouterLink>
    </nav>
    <EmptyState message="El panel de control estará disponible próximamente." />
  </main>
</template>

<style scoped>
.dashboard {
  max-width: 960px;
  margin: 0 auto;
  padding: var(--space-4);
}

.dashboard-nav {
  margin: var(--space-3) 0;
  display: flex;
  gap: var(--space-3);
}

.dashboard-nav a {
  color: var(--color-primary);
  font-weight: 600;
}
</style>
