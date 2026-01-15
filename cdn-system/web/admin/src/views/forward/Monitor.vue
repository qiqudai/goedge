<template>
  <div class="app-container forward-monitor">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="activeTopTab" class="custom-tabs" @tab-change="handleTopTab">
        <el-tab-pane label="转发列表" name="list" />
        <el-tab-pane label="默认设置" name="default" />
        <el-tab-pane label="实时监控" name="monitor">
          <el-tabs v-model="activeTab" class="monitor-inner-tabs" @tab-change="handleInnerTab">
            <el-tab-pane label="带宽流量" name="traffic">
              <div class="filter-container">
                <el-input v-model="query.keyword" placeholder="端口检索 (如: 88/TCP)" style="width: 200px; margin-right: 12px;" />
                <el-button-group style="margin-right: 12px;">
                  <el-button :type="range === '1h' ? 'primary' : 'default'" @click="setRange('1h')">1h</el-button>
                  <el-button :type="range === '6h' ? 'primary' : 'default'" @click="setRange('6h')">6h</el-button>
                  <el-button :type="range === '24h' ? 'primary' : 'default'" @click="setRange('24h')">24h</el-button>
                </el-button-group>
                <el-button type="primary" @click="reload">刷新</el-button>
              </div>
              <el-row :gutter="20">
                <el-col :span="12">
                  <div class="chart-box">
                    <div class="chart-header">带宽占用 (Mbps)</div>
                    <div id="bandwidthChart" class="chart-body"></div>
                  </div>
                </el-col>
                <el-col :span="12">
                  <div class="chart-box">
                    <div class="chart-header">流量统计 (GB)</div>
                    <div id="trafficChart" class="chart-body"></div>
                  </div>
                </el-col>
              </el-row>
            </el-tab-pane>

            <el-tab-pane label="端口排行" name="ranking">
              <div class="filter-container">
                <el-button type="primary" size="small" @click="reloadRanking">即时刷新</el-button>
              </div>
              <AppTable :data="ranking" border fit persist-key="forward-ranking-list">
                <el-table-column prop="rank" label="排名" width="80" align="center">
                    <template #default="{ $index }">
                        <el-tag :type="$index < 3 ? 'danger' : 'info'" effect="dark" round>{{ $index + 1 }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="port" label="业务端口" min-width="150" />
                <el-table-column prop="connections" label="当前连接数" width="150" align="right" sortable />
                <el-table-column prop="traffic" label="累计流量" width="150" align="right" sortable />
              </AppTable>
            </el-tab-pane>
          </el-tabs>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import * as echarts from 'echarts'
import request from '@/utils/request'

const router = useRouter()
const activeTopTab = ref('monitor')
const activeTab = ref('traffic')
const range = ref('1h')
const query = reactive({ keyword: '' })
const ranking = ref([])
const trafficLoading = ref(false)
const rankingLoading = ref(false)

let bandwidthChart = null
let trafficChart = null

const handleTopTab = (name) => {
  const map = {
    list: '/forward/list',
    default: '/forward/default',
    monitor: '/forward/monitor'
  }
  const path = map[name]
  if (name === 'monitor') {
    reloadAll()
    return
  }
  if (path) router.push(path)
}

const getChartColors = () => {
  const isDark = document.documentElement.getAttribute('data-theme') === 'dark'
  return {
    axisLabel: isDark ? '#b0b6c3' : '#606266',
    axisLine: isDark ? '#3a3f47' : '#e4e7ed',
    splitLine: isDark ? '#343a43' : '#eef1f6'
  }
}

const buildChartOption = (title, color, data) => {
  const colors = getChartColors()
  return {
  title: { show: false },
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', top: '5%', bottom: '3%', containLabel: true },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: data.times,
    axisLine: { lineStyle: { color: colors.axisLine } },
    axisLabel: { color: colors.axisLabel }
  },
  yAxis: {
    type: 'value',
    axisLine: { lineStyle: { color: colors.axisLine } },
    axisLabel: { color: colors.axisLabel },
    splitLine: { lineStyle: { color: colors.splitLine } }
  },
  series: [{
    name: title,
    type: 'line',
    smooth: true,
    showSymbol: false,
    itemStyle: { color },
    areaStyle: {
      color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: color + '40' },
        { offset: 1, color: color + '05' }
      ])
    },
    data: data.values
  }]
  }
}

const ensureCharts = () => {
  const bwDom = document.getElementById('bandwidthChart')
  const trDom = document.getElementById('trafficChart')
  if (!bwDom || !trDom) return false
  if (!bandwidthChart) bandwidthChart = echarts.init(bwDom)
  if (!trafficChart) trafficChart = echarts.init(trDom)
  return true
}

const updateCharts = (payload) => {
  if (!ensureCharts()) return
  const times = payload?.x_axis || []
  const bwValues = payload?.bandwidth || []
  const trValues = payload?.traffic || []
  bandwidthChart.setOption(buildChartOption('带宽', '#409eff', { times, values: bwValues }), true)
  trafficChart.setOption(buildChartOption('流量', '#67c23a', { times, values: trValues }), true)
}

const loadTraffic = async () => {
  trafficLoading.value = true
  try {
    const res = await request.get('/forward/traffic', {
      params: { range: range.value, keyword: query.keyword }
    })
    if (res.code === 0) {
      await nextTick()
      updateCharts(res.data || {})
    }
  } finally {
    trafficLoading.value = false
  }
}

const reload = () => loadTraffic()

const loadRanking = async () => {
  rankingLoading.value = true
  try {
    const res = await request.get('/forward/ranking', {
      params: { range: range.value }
    })
    ranking.value = res.data?.list || []
  } finally {
    rankingLoading.value = false
  }
}

const reloadRanking = () => loadRanking()

const handleInnerTab = (name) => {
  if (name === 'traffic') {
    reload()
    return
  }
  if (name === 'ranking') {
    reloadRanking()
  }
}

const setRange = (val) => { range.value = val; reload() }

const reloadAll = () => {
  reload()
  reloadRanking()
}

const handleResize = () => {
    bandwidthChart?.resize()
    trafficChart?.resize()
}

onMounted(() => {
    reloadAll()
    window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    bandwidthChart?.dispose()
    trafficChart?.dispose()
})
</script>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
.custom-tabs :deep(.el-tabs__item) { font-weight: 600; }
.monitor-inner-tabs { margin-top: 10px; }
.filter-container { margin-bottom: 20px; display: flex; align-items: center; }
.chart-box { background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 4px; padding: 15px; margin-bottom: 20px; }
.chart-header { font-size: 14px; font-weight: 600; color: var(--muted-text); margin-bottom: 15px; }
.chart-body { height: 300px; }

:root[data-theme="dark"] .forward-monitor :deep(.el-icon) {
  color: var(--muted-text);
}
</style>
