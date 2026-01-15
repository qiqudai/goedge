<template>
  <div class="app-container">
    <div class="usage-header">
      <div class="usage-title">用量查询</div>
      <div class="usage-actions">
        <el-button-group class="range-button-group">
          <el-button size="small" :type="range === 'today' ? 'primary' : 'default'" @click="setRange('today')">今天</el-button>
          <el-button size="small" :type="range === 'yesterday' ? 'primary' : 'default'" @click="setRange('yesterday')">昨天</el-button>
          <el-button size="small" :type="range === '7days' ? 'primary' : 'default'" @click="setRange('7days')">近7天</el-button>
          <el-button size="small" :type="range === '30days' ? 'primary' : 'default'" @click="setRange('30days')">30天</el-button>
        </el-button-group>
        <el-button link type="primary" @click="loadUsage">刷新</el-button>
      </div>
    </div>

    <div class="summary-row">
      <div class="summary-card">
        <div class="summary-label">总流量</div>
        <div class="summary-value">{{ summary.total }} {{ summary.unit }}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">峰值</div>
        <div class="summary-value">{{ summary.peak }} {{ summary.unit }}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">平均值</div>
        <div class="summary-value">{{ summary.avg }} {{ summary.unit }}</div>
      </div>
    </div>

    <div class="chart-wrapper">
      <div ref="usageChartRef" class="chart" />
    </div>

    <el-table :data="tableData" border v-loading="loading" style="width: 100%;">
      <el-table-column prop="time" label="时间" min-width="200" />
      <el-table-column prop="value" label="用量" min-width="160">
        <template #default="{ row }">{{ row.value }} {{ summary.unit }}</template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { nextTick, onMounted, ref } from 'vue'
import request from '@/utils/request'
import * as echarts from 'echarts'

const range = ref('today')
const loading = ref(false)
const tableData = ref([])
const usageChartRef = ref(null)
let usageChart = null

const summary = ref({
  total: 0,
  avg: 0,
  peak: 0,
  unit: 'MB'
})

const buildChart = (xAxis, series, unit) => {
  if (!usageChartRef.value) return
  if (!usageChart) {
    usageChart = echarts.init(usageChartRef.value)
  }
  usageChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 24, top: 20, bottom: 40 },
    xAxis: {
      type: 'category',
      data: xAxis,
      axisLabel: { color: '#909399' }
    },
    yAxis: {
      type: 'value',
      name: unit,
      axisLabel: { color: '#909399' }
    },
    series: [
      {
        name: '流量',
        type: 'line',
        smooth: true,
        data: series,
        areaStyle: {}
      }
    ]
  })
}

const setRange = (val) => {
  range.value = val
  loadUsage()
}

const loadUsage = async () => {
  loading.value = true
  try {
    const res = await request.get('/usage', { params: { range: range.value } })
    if (res.code === 0) {
      const data = res.data || {}
      summary.value = {
        total: data.total ?? 0,
        avg: data.avg ?? 0,
        peak: data.peak ?? 0,
        unit: data.unit || 'MB'
      }
      tableData.value = data.list || []
      await nextTick()
      buildChart(data.x_axis || [], data.values || [], summary.value.unit)
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadUsage()
  window.addEventListener('resize', () => {
    if (usageChart) usageChart.resize()
  })
})
</script>

<style scoped>
.usage-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.usage-title {
  font-size: 18px;
  font-weight: 600;
}

.usage-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.summary-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.summary-card {
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 12px 16px;
  background: var(--card-bg);
}

.summary-label {
  color: var(--muted-text);
  font-size: 12px;
  margin-bottom: 6px;
}

.summary-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.chart-wrapper {
  margin-bottom: 16px;
}

.chart {
  width: 100%;
  height: 320px;
}
</style>
