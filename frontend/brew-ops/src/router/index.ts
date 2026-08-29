import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    // Every route is protected by default — only routes explicitly marked
    // public (currently just /login) skip the auth check below. This way
    // adding a new route can never accidentally forget to protect it.
    public?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
    },
    {
      path: '/products',
      name: 'products',
      component: () => import('../views/ProductsView.vue'),
    },
    {
      path: '/products/trash',
      name: 'products-trash',
      component: () => import('../views/ProductsTrashView.vue'),
    },
    {
      path: '/pos',
      name: 'pos',
      component: () => import('../views/PointOfSaleView.vue'),
    },
    {
      path: '/sales',
      name: 'sales-history',
      component: () => import('../views/SalesHistoryView.vue'),
    },
    {
      path: '/marketing',
      name: 'marketing',
      component: () => import('../views/MarketingView.vue'),
    },
    {
      path: '/marketing/trash',
      name: 'marketing-trash',
      component: () => import('../views/MarketingTrashView.vue'),
    },
  ],
})

router.beforeEach((to) => {
  const authStore = useAuthStore()

  if (to.meta.public) {
    // Already logged in and navigating to /login directly — go straight
    // to the dashboard instead of showing the form again.
    if (to.name === 'login' && authStore.isAuthenticated) {
      return { path: '/' }
    }
    return true
  }

  if (!authStore.isAuthenticated) {
    // Preserve the originally requested route so a successful login lands
    // the user there instead of always on the dashboard.
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  return true
})

export default router
