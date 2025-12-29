<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="activeTopTab" class="custom-tabs" @tab-change="handleTopTab">
        <el-tab-pane label="转发列表" name="list" />
        <el-tab-pane label="默认设置" name="default" />
        <el-tab-pane label="实时监控" name="monitor">
          <el-tabs v-model="activeTab" class="monitor-inner-tabs">
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

const router = useRouter()
const activeTopTab = ref('monitor')
const activeTab = ref('traffic')
const range = ref('1h')
const query = reactive({ keyword: '' })
const ranking = ref([])

let bandwidthChart = null
let trafficChart = null

const handleTopTab = (name) => {
  const map = {
    list: '/forward/list',
    default: '/forward/default',
    monitor: '/forward/monitor'
  }
  const path = map[name]
  if (path && name !== 'monitor') router.push(path)
}

const buildChartOption = (title, color, data) => ({
  title: { show: false },
  tooltip: { trigger: 'axis' },
  grid: { left: '3%', right: '4%', top: '5%', bottom: '3%', containLabel: true },
  xAxis: { type: 'category', boundaryGap: false, data: data.times },
  yAxis: { type: 'value' },
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
})

const initCharts = () => {
  if (!bandwidthChart) bandwidthChart = echarts.init(document.getElementById('bandwidthChart'))
  if (!trafficChart) trafficChart = echarts.init(document.getElementById('trafficChart'))

  const data = {
    times: Array.from({ length: 24 }, (_, i) => `${i}:00`),
    bwValues: Array.from({ length: 24 }, () => (Math.random() * 100).toFixed(2)),
    trValues: Array.from({ length: 24 }, () => (Math.random() * 10).toFixed(2))
  }

  bandwidthChart.setOption(buildChartOption('带宽', '#409eff', { times: data.times, values: data.bwValues }))
  trafficChart.setOption(buildChartOption('流量', '#67c23a', { times: data.times, values: data.trValues }))
}

const reload = () => { nextTick(initCharts) }

const loadRanking = () => {
    ranking.value = Array.from({ length: 10 }, (_, i) => ({
        port: `${8000 + i}/TCP`,
        connections: Math.floor(Math.random() * 1000),
        traffic: (Math.random() * 50).toFixed(2) + ' GB'
    }))
}

const reloadRanking = () => loadRanking()

const setRange = (val) => { range.value = val; reload() }

const handleResize = () => {
    bandwidthChart?.resize()
    trafficChart?.resize()
}

onMounted(() => {
    initCharts()
    loadRanking()
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
.chart-box { background: #fcfcfc; border: 1px solid #f0f0f0; border-radius: 4px; padding: 15px; margin-bottom: 20px; }
.chart-header { font-size: 14px; font-weight: 600; color: #606266; margin-bottom: 15px; }
.chart-body { height: 300px; }
</style>
