<template>
  <div class="app-container">
    <el-tabs v-model="activeTopTab" class="site-tabs" @tab-click="handleTopTab">
      <el-tab-pane label="转发列表" name="list" />
      <el-tab-pane label="分组设置" name="groups" />
      <el-tab-pane label="默认设置" name="default" />
      <el-tab-pane label="实时监控" name="monitor" />
    </el-tabs>
    <el-tabs v-model="activeTab" class="monitor-tabs">
      <el-tab-pane label="带宽流量" name="traffic">
        <div class="filter-container">
          <el-input v-model="query.keyword" placeholder="输入端口, 如88/TCP 99/UDP" style="width: 240px;" />
          <el-button-group>
            <el-button :type="range === '1h' ? 'primary' : 'default'" @click="setRange('1h')">近1小时</el-button>
            <el-button :type="range === '6h' ? 'primary' : 'default'" @click="setRange('6h')">近6小时</el-button>
            <el-button :type="range === '12h' ? 'primary' : 'default'" @click="setRange('12h')">近12小时</el-button>
            <el-button :type="range === 'custom' ? 'primary' : 'default'" @click="setRange('custom')">自定义</el-button>
          </el-button-group>
          <el-button type="primary" @click="reload">查询</el-button>
        </div>
        <el-row :gutter="16">
          <el-col :span="12">
            <div class="chart-title">带宽</div>
            <div id="bandwidthChart" class="chart"></div>
          </el-col>
          <el-col :span="12">
            <div class="chart-title">流量</div>
            <div id="trafficChart" class="chart"></div>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane label="端口排行" name="ranking">
        <div class="filter-container">
          <el-button-group>
            <el-button :type="rankRange === '10m' ? 'primary' : 'default'" @click="setRankRange('10m')">10分钟实时</el-button>
            <el-button :type="rankRange === '30m' ? 'primary' : 'default'" @click="setRankRange('30m')">近30分钟</el-button>
            <el-button :type="rankRange === '1h' ? 'primary' : 'default'" @click="setRankRange('1h')">近1小时</el-button>
          </el-button-group>
          <el-button type="primary" @click="reloadRanking">刷新</el-button>
        </div>
        <el-table :data="ranking" border size="small">
          <el-table-column prop="rank" label="排行" width="80" />
          <el-table-column prop="port" label="端口" />
          <el-table-column prop="connections" label="连接数" sortable />
          <el-table-column prop="traffic" label="流量" sortable />
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import * as echarts from 'echarts'

const router = useRouter()
const activeTopTab = ref('monitor')
const handleTopTab = tab => {
  const name = typeof tab === 'string' ? tab : tab?.paneName
  const map = {
    list: '/forward/list',
    groups: '/forward/groups',
    default: '/forward/default',
    monitor: '/forward/monitor'
  }
  const path = map[name]
  if (path) {
    router.push(path)
  }
}
const activeTab = ref('traffic')
const range = ref('1h')
const rankRange = ref('10m')
const query = reactive({ keyword: '' })
const ranking = ref([])

let bandwidthChart = null
let trafficChart = null

const buildSeries = (name, color) => ({
  name,
  type: 'line',
  smooth: true,
  symbol: 'circle',
  symbolSize: 4,
  itemStyle: { color },
  lineStyle: { color },
  areaStyle: {
    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
      { offset: 0, color: 'rgba(64,158,255,0.25)' },
      { offset: 1, color: 'rgba(64,158,255,0.05)' }
    ])
  }
})

const renderCharts = () => {
  const times = Array.from({ length: 12 }).map((_, i) => `12-22 19:${String(i * 2).padStart(2, '0')}`)
  const bandwidth = times.map(() => Number((Math.random() * 5).toFixed(2)))
  const traffic = times.map(() => Number((Math.random() * 3).toFixed(2)))

  if (!bandwidthChart) {
    bandwidthChart = echarts.init(document.getElementById('bandwidthChart'))
  }
  if (!trafficChart) {
    trafficChart = echarts.init(document.getElementById('trafficChart'))
  }

  bandwidthChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: times },
    yAxis: { type: 'value' },
    series: [{ ...buildSeries('带宽', '#409eff'), data: bandwidth }]
  })

  trafficChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: times },
    yAxis: { type: 'value' },
    series: [{ ...buildSeries('流量', '#67c23a'), data: traffic }]
  })
}

const loadRanking = () => {
  ranking.value = Array.from({ length: 8 }).map((_, idx) => ({
    rank: idx + 1,
    port: ['88/tcp', '99/udp', '443/tcp', '10001/tcp'][idx % 4],
    connections: Math.floor(Math.random() * 5000),
    traffic: `${(Math.random() * 12).toFixed(2)} MB`
  }))
}

const setRange = val => {
  range.value = val
  reload()
}

const setRankRange = val => {
  rankRange.value = val
  reloadRanking()
}

const reload = () => {
  nextTick(renderCharts)
}

const reloadRanking = () => {
  loadRanking()
}

onMounted(() => {
  renderCharts()
  loadRanking()
  window.addEventListener('resize', () => {
    bandwidthChart && bandwidthChart.resize()
    trafficChart && trafficChart.resize()
  })
})
</script>

<style scoped>
.filter-container {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
  margin-bottom: 12px;
}
.chart {
  height: 260px;
}
.chart-title {
  margin-bottom: 8px;
  font-size: 13px;
  color: #606266;
}
</style>

