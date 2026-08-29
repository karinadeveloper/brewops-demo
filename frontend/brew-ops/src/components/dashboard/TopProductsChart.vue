<script setup lang="ts">
import { computed } from 'vue'
import { Bar } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, BarElement, Tooltip, Legend, type TooltipItem } from 'chart.js'
import type { TopProduct } from '../../stores/dashboard'
import { centsToPesos, formatCentsAsPesos } from '../../utils/money'

ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip, Legend)

const props = defineProps<{ products: TopProduct[] }>()

const chartData = computed(() => ({
  labels: props.products.map((p) => p.name),
  datasets: [
    {
      label: 'Ingresos',
      data: props.products.map((p) => centsToPesos(p.revenue_cents)),
      backgroundColor: 'hsla(28, 85%, 55%, 0.75)',
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: TooltipItem<'bar'>) => formatCentsAsPesos(Math.round((ctx.parsed.y ?? 0) * 100)),
      },
    },
  },
  scales: { y: { beginAtZero: true } },
}
</script>

<template>
  <div class="top-products-chart">
    <Bar
      :data="chartData"
      :options="chartOptions"
    />
  </div>
</template>

<style scoped>
.top-products-chart {
  height: 16rem;
}
</style>
