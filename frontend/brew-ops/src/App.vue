<script setup lang="ts">
// Registered once, here, so automatic sale sync runs for the whole app's
// lifetime regardless of which view is currently active — not just while
// SalesHistoryView happens to be mounted.
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useSaleSync } from './composables/useSaleSync'
import AppNav from './components/shared/AppNav.vue'

const route = useRoute()
const { t } = useI18n()

const dateFormatter = new Intl.DateTimeFormat('es-MX', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'America/Mexico_City',
})

useSaleSync((sale, message) => {
  // A business rejection (e.g. insufficient stock) needs a clear, visible
  // alert, since it can happen entirely in the background — the user may
  // not be looking at the sales history when connectivity returns.
  // window.alert is blocking and guaranteed visible regardless of the
  // current view — a toast/notification system would be a nicer future
  // upgrade, but none exists in this codebase yet.
  window.alert(
    t('app.saleRejectedAlert', { date: dateFormatter.format(new Date(sale.createdAt)), message }),
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
