<script setup lang="ts">
import { onMounted } from 'vue'
import EmptyState from '../components/shared/EmptyState.vue'
import SkeletonList from '../components/shared/SkeletonList.vue'
import MetricCard from '../components/dashboard/MetricCard.vue'
import PeriodSelector from '../components/dashboard/PeriodSelector.vue'
import LowStockAlerts from '../components/dashboard/LowStockAlerts.vue'
import RevenueChart from '../components/dashboard/RevenueChart.vue'
import TopProductsChart from '../components/dashboard/TopProductsChart.vue'
import { useDashboardStore } from '../stores/dashboard'
import { formatCentsAsPesos } from '../utils/money'
import type { Period } from '../utils/period'

const dashboard = useDashboardStore()

onMounted(() => {
  void dashboard.loadAll()
})

function onPeriodChange(period: Period) {
  dashboard.setPeriod(period)
}
</script>

<template>
  <main class="dashboard">
    <header class="dashboard-header">
      <h1>Dashboard</h1>
      <PeriodSelector
        :model-value="dashboard.period"
        @update:model-value="onPeriodChange"
      />
    </header>

    <section
      class="metric-cards"
      aria-label="Métricas resumen"
    >
      <MetricCard
        title="Ingresos del período"
        :value="formatCentsAsPesos(dashboard.revenueTotalCents)"
        :is-loading="dashboard.isLoadingRevenue"
        :error="dashboard.revenueError"
        @retry="dashboard.loadRevenue"
      />
      <MetricCard
        title="Ventas del período"
        :value="String(dashboard.salesCount)"
        :is-loading="dashboard.isLoadingSalesCount"
        :error="dashboard.salesCountError"
        @retry="dashboard.loadSalesCount"
      />
      <MetricCard
        title="Valor de inventario"
        :value="formatCentsAsPesos(dashboard.inventoryValueCents)"
        :is-loading="dashboard.isLoadingInventoryValue"
        :error="dashboard.inventoryValueError"
        @retry="dashboard.loadInventoryValue"
      />
    </section>

    <LowStockAlerts
      :items="dashboard.lowStockItems"
      :is-loading="dashboard.isLoadingLowStock"
      :error="dashboard.lowStockError"
      @retry="dashboard.loadLowStock"
    />

    <section
      class="chart-section"
      aria-label="Ingresos en el tiempo"
    >
      <h2>Ingresos en el tiempo</h2>
      <SkeletonList
        v-if="dashboard.isLoadingRevenue"
        :rows="1"
      />
      <EmptyState
        v-else-if="dashboard.revenueError"
        title="No se pudo cargar la gráfica"
        message="Ocurrió un error al conectar con el servidor."
      >
        <template #action>
          <button
            type="button"
            class="btn"
            @click="dashboard.loadRevenue"
          >
            Reintentar
          </button>
        </template>
      </EmptyState>
      <EmptyState
        v-else-if="dashboard.revenuePoints.length === 0"
        message="No hay ventas registradas en este período."
      />
      <RevenueChart
        v-else
        :points="dashboard.revenuePoints"
      />
    </section>

    <section
      class="chart-section"
      aria-label="Productos más vendidos"
    >
      <h2>Productos más vendidos</h2>
      <SkeletonList
        v-if="dashboard.isLoadingTopProducts"
        :rows="1"
      />
      <EmptyState
        v-else-if="dashboard.topProductsError"
        title="No se pudo cargar la gráfica"
        message="Ocurrió un error al conectar con el servidor."
      >
        <template #action>
          <button
            type="button"
            class="btn"
            @click="dashboard.loadTopProducts"
          >
            Reintentar
          </button>
        </template>
      </EmptyState>
      <EmptyState
        v-else-if="dashboard.topProducts.length === 0"
        message="No hay ventas registradas en este período."
      />
      <TopProductsChart
        v-else
        :products="dashboard.topProducts"
      />
    </section>
  </main>
</template>

<style scoped>
.dashboard {
  max-width: 1100px;
  margin: 0 auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.dashboard-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.metric-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
  gap: var(--space-3);
}

.chart-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.btn {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
}
</style>
