<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '../../stores/auth'

const authStore = useAuthStore()
const isMobileMenuOpen = ref(false)

// Trash views (products, marketing) are deliberately not top-level links
// here — each is one "Ver papelera" click away from its own list view
// (ProductsView, MarketingView), which scales better than growing this
// list by one entry per resource that ever gets a trash view.
const links = [
  { to: '/', label: 'Dashboard' },
  { to: '/pos', label: 'Punto de venta' },
  { to: '/sales', label: 'Historial de ventas' },
  { to: '/products', label: 'Productos' },
  { to: '/marketing', label: 'Marketing' },
]

function closeMobileMenu() {
  isMobileMenuOpen.value = false
}

function handleLogout() {
  closeMobileMenu()
  authStore.logout()
}
</script>

<template>
  <header class="app-nav-topbar">
    <span class="app-nav-brand">BrewOps</span>
    <button
      type="button"
      class="app-nav-toggle"
      :aria-expanded="isMobileMenuOpen"
      aria-controls="app-nav-menu"
      aria-label="Abrir menú de navegación"
      @click="isMobileMenuOpen = !isMobileMenuOpen"
    >
      ☰
    </button>
  </header>

  <nav
    id="app-nav-menu"
    class="app-nav"
    :class="{ 'app-nav--open': isMobileMenuOpen }"
    aria-label="Navegación principal"
  >
    <span class="app-nav-brand app-nav-brand--sidebar">BrewOps</span>
    <ul class="app-nav-list">
      <li
        v-for="link in links"
        :key="link.to"
      >
        <RouterLink
          :to="link.to"
          class="app-nav-link"
          @click="closeMobileMenu"
        >
          {{ link.label }}
        </RouterLink>
      </li>
    </ul>
    <button
      type="button"
      class="app-nav-logout"
      @click="handleLogout"
    >
      Cerrar sesión
    </button>
  </nav>

  <div
    v-if="isMobileMenuOpen"
    class="app-nav-backdrop"
    role="presentation"
    @click="closeMobileMenu"
  />
</template>

<style scoped>
.app-nav-topbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: var(--topbar-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-3);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  z-index: 20;
}

.app-nav-brand {
  font-weight: 700;
  color: var(--color-primary);
}

.app-nav-brand--sidebar {
  display: none;
}

.app-nav-toggle {
  font-size: 1.25rem;
  line-height: 1;
  background: none;
  border: none;
  padding: var(--space-2);
  color: var(--color-text);
}

.app-nav {
  position: fixed;
  top: var(--topbar-height);
  left: 0;
  bottom: 0;
  width: var(--nav-width);
  max-width: 80vw;
  background: var(--color-surface);
  border-right: 1px solid var(--color-border);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  transform: translateX(-100%);
  transition: transform 0.2s ease;
  z-index: 25;
  overflow-y: auto;
}

.app-nav--open {
  transform: translateX(0);
}

.app-nav-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  list-style: none;
  padding: 0;
  margin: 0;
}

.app-nav-link {
  display: block;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  color: var(--color-text);
}

.app-nav-link.router-link-exact-active {
  background: var(--color-surface-alt);
  color: var(--color-primary);
  font-weight: 600;
}

.app-nav-logout {
  margin-top: auto;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  text-align: left;
  color: var(--color-text);
}

.app-nav-backdrop {
  position: fixed;
  inset: 0;
  background: hsla(220, 15%, 15%, 0.4);
  z-index: 22;
}

@media (min-width: 768px) {
  .app-nav-topbar {
    display: none;
  }

  .app-nav {
    top: 0;
    transform: none;
    z-index: 15;
  }

  .app-nav-brand--sidebar {
    display: block;
    font-size: 1.25rem;
    margin-bottom: var(--space-2);
  }

  .app-nav-backdrop {
    display: none;
  }
}
</style>
