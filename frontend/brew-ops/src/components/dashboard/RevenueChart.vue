<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Filler,
  Tooltip,
  Legend,
  type TooltipItem,
} from 'chart.js'
import type { RevenuePoint } from '../../stores/dashboard'
import { centsToPesos, formatCentsAsPesos } from '../../utils/money'
import { formatDayLabel } from '../../utils/period'

// Filler is required for the dataset's fill: true below to actually draw
// the area under the line — without registering it, Chart.js silently
// skips the fill and only logs a console warning, easy to miss since
// nothing visibly breaks.
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, Tooltip, Legend)

const { t } = useI18n()
const props = defineProps<{ points: RevenuePoint[] }>()

const chartData = computed(() => ({
  labels: props.points.map((p) => (p.day ? formatDayLabel(p.day) : '')),
  datasets: [
    {
      label: t('charts.revenueLabel'),
      data: props.points.map((p) => centsToPesos(p.total_cents)),
      borderColor: 'hsla(160, 60%, 40%, 1)',
      backgroundColor: 'hsla(160, 60%, 40%, 0.15)',
      tension: 0.3,
      fill: true,
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
        label: (ctx: TooltipItem<'line'>) => formatCentsAsPesos(Math.round((ctx.parsed.y ?? 0) * 100)),
      },
    },
  },
  scales: { y: { beginAtZero: true } },
}
</script>

<template>
  <div class="revenue-chart">
    <Line
      :data="chartData"
      :options="chartOptions"
    />
  </div>
</template>

<style scoped>
.revenue-chart {
  height: 16rem;
}
</style>
