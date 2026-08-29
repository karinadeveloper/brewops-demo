<script setup lang="ts">
// Registered once, here, so automatic sale sync runs for the whole app's
// lifetime regardless of which view is currently active — not just while
// SalesHistoryView happens to be mounted.
import { useRoute } from 'vue-router'
import { useSaleSync } from './composables/useSaleSync'
import AppNav from './components/shared/AppNav.vue'

const route = useRoute()

const dateFormatter = new Intl.DateTimeFormat('es-MX', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'America/Mexico_City',
})

useSaleSync((sale, message) => {
  // Per CLAUDE.md's Sync UX contract: a business rejection needs a clear,
  // visible alert, since it can happen entirely in the background (the
  // user may not be looking at the sales history when connectivity
  // returns). window.alert is blocking and guaranteed visible regardless
  // of the current view — a toast/notification system would be a nicer
  // future upgrade, but none exists in this codebase yet.
  window.alert(
    `No se pudo aplicar la venta del ${dateFormatter.format(new Date(sale.createdAt))}: ${message}`,
  )
})
</script>

<template>
  <router-view v-if="route.meta.public" />
  <div
    v-else
    class="app-shell"
  >
    <AppNav />
    <div class="app-content">
      <router-view />
    </div>
  </div>
</template>

<style scoped>
.app-content {
  padding-top: var(--topbar-height);
}

@media (min-width: 768px) {
  .app-content {
    padding-top: 0;
    margin-left: var(--nav-width);
  }
}
</style>
