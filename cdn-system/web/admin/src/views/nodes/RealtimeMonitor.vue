<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" class="monitor-tabs">
      <el-tab-pane label="资源排行" name="ranking" />
      <el-tab-pane label="监控指标" name="metrics" />
      <el-tab-pane label="节点流量" name="traffic" />
    </el-tabs>

    <div v-if="activeTab === 'ranking'">
      <div class="monitor-toolbar">
        <div class="toolbar-row">
          <span class="toolbar-label">指标</span>
          <el-radio-group v-model="ranking.metric">
            <el-radio-button value="bandwidth">带宽</el-radio-button>
            <el-radio-button value="connection">连接</el-radio-button>
            <el-radio-button value="load">负载</el-radio-button>
            <el-radio-button value="disk">硬盘</el-radio-button>
          </el-radio-group>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">时间</span>
          <el-radio-group v-model="ranking.window">
            <el-radio-button value="1m">1分钟</el-radio-button>
            <el-radio-button value="5m">5分钟</el-radio-button>
            <el-radio-button value="30m">30分钟</el-radio-button>
            <el-radio-button value="1h">1小时</el-radio-button>
          </el-radio-group>
        </div>
        <el-button type="primary" class="refresh-button" style="width: 96px;" @click="refreshRanking">刷新</el-button>
      </div>

      <AppTable :data="ranking.list" :loading="rankingLoading" border persist-key="node-ranking">
        <el-table-column prop="rank" label="排行" width="80" align="center" />
        <el-table-column prop="node" label="节点" min-width="160" />
        <el-table-column prop="nic" label="网卡" min-width="120" />
        <el-table-column prop="out" label="出站带宽" min-width="140" />
        <el-table-column prop="in" label="入站带宽" min-width="140" />
      </AppTable>
    </div>

    <div v-if="activeTab === 'metrics'">
      <div class="monitor-toolbar">
        <div class="toolbar-row">
          <span class="toolbar-label">指标</span>
          <el-radio-group v-model="metrics.metric">
            <el-radio-button value="bandwidth">带宽</el-radio-button>
            <el-radio-button value="connection">连接</el-radio-button>
            <el-radio-button value="load">负载</el-radio-button>
            <el-radio-button value="disk">硬盘</el-radio-button>
          </el-radio-group>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">Time</span>
          <el-radio-group v-model="metrics.window">
            <el-radio-button value="1h">1h</el-radio-button>
            <el-radio-button value="6h">6h</el-radio-button>
            <el-radio-button value="12h">12h</el-radio-button>
            <el-radio-button value="custom">Custom</el-radio-button>
          </el-radio-group>
          <el-date-picker
            v-if="metrics.window === 'custom'"
            v-model="metrics.timeRange"
            type="datetimerange"
            range-separator="to"
            start-placeholder="Start time"
            end-placeholder="End time"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            class="time-range"
            style="width: 320px;"
          />
        </div>
        <el-button type="primary" class="refresh-button" style="width: 96px;" @click="refreshMetrics">刷新</el-button>
      </div>

      <AppTable :data="metrics.list" :loading="metricsLoading" border persist-key="node-metrics">
        <el-table-column prop="time" label="Time" min-width="160" />
        <el-table-column prop="value" label="Value" min-width="120" />
      </AppTable>
    </div>

    <div v-if="activeTab === 'traffic'">
      <div class="monitor-toolbar">
        <div class="toolbar-row">
          <span class="toolbar-label">类型</span>
          <el-checkbox v-model="traffic.out">出站流量</el-checkbox>
          <el-checkbox v-model="traffic.in">入站流量</el-checkbox>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">Time</span>
          <el-radio-group v-model="traffic.window">
            <el-radio-button value="1d">1d</el-radio-button>
            <el-radio-button value="7d">7d</el-radio-button>
            <el-radio-button value="30d">30d</el-radio-button>
            <el-radio-button value="custom">Custom</el-radio-button>
          </el-radio-group>
          <el-date-picker
            v-if="traffic.window === 'custom'"
            v-model="traffic.timeRange"
            type="datetimerange"
            range-separator="to"
            start-placeholder="Start time"
            end-placeholder="End time"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            class="time-range"
            style="width: 320px;"
            @change="handleTrafficTimeRangeChange"
          />
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">节点</span>
          <el-select v-model="traffic.node" style="width: 220px;" filterable placeholder="选择节点" @change="refreshTraffic">
            <el-option label="全部节点" value="all" />
            <el-option v-for="item in nodeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">排除网卡</span>
          <el-input v-model="traffic.exclude" placeholder="排除网卡，多个网卡用空格分隔" style="width: 320px;" @change="refreshTraffic" />
        </div>
        <el-button type="primary" class="refresh-button" style="width: 96px;" @click="refreshTraffic">刷新</el-button>
      </div>

      <div class="chart-container">
        <div class="chart-title">节点流量</div>
        <div id="trafficChart" class="traffic-chart"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, nextTick, watch } from 'vue'
import request from '@/utils/request'
import * as echarts from 'echarts'

const activeTab = ref('ranking')
const trafficChartRef = ref(null)
let myChart = null
const rankingLoading = ref(false)
const metricsLoading = ref(false)

const ranking = reactive({
  metric: 'bandwidth',
  window: '1m',
  list: []
})

const metrics = reactive({
  metric: 'bandwidth',
  window: '1h',
  timeRange: [],
  list: []
})

const traffic = reactive({
  out: true,
  in: true,
  window: '30d',
  timeRange: [],
  node: 'all',
  exclude: ''
})

const nodeOptions = ref([])

onMounted(() => {
  fetchNodes()
  refreshRanking()
})

const fetchNodes = () => {
  request.get('/nodes', { params: { pageSize: 1000 } }).then(res => {
    if ((res.code === 0 || res.code === 200) && res.data && res.data.list) {
      nodeOptions.value = res.data.list.map(node => ({
        label: node.name,
        value: node.id
      }))
    }
  })
}

const refreshRanking = async () => {
  if (activeTab.value !== 'ranking') return
  rankingLoading.value = true
  try {
    const res = await request.get('/stats/node_ranking', { params: { metric: ranking.metric, window: ranking.window } })
    ranking.list = res.data?.list || []
  } finally {
    rankingLoading.value = false
  }
}

const refreshMetrics = async () => {
  if (activeTab.value !== 'metrics') return
  if (metrics.window === 'custom') {
    if (!metrics.timeRange?.[0] || !metrics.timeRange?.[1]) {
      metrics.list = []
      return
    }
  }
  metricsLoading.value = true
  try {
    const res = await request.get('/stats/node_metrics', {
      params: {
        metric: metrics.metric,
        window: metrics.window,
        start_time: metrics.timeRange?.[0],
        end_time: metrics.timeRange?.[1]
      }
    })
    metrics.list = res.data?.list || []
  } finally {
    metricsLoading.value = false
  }
}

// Chart Logic
const initChart = () => {
  const chartDom = document.getElementById('trafficChart')
  if (!chartDom) return
  if (myChart) {
      myChart.dispose()
  }
  myChart = echarts.init(chartDom)
  refreshTraffic() // Load data initially
  window.addEventListener('resize', () => myChart && myChart.resize())
}

const refreshTraffic = () => {
  if (!myChart) return // Wait for init
  if (traffic.window === 'custom' && (!traffic.timeRange?.[0] || !traffic.timeRange?.[1])) {
    return
  }
  
  myChart.showLoading()
  request.get('/stats/node_traffic', {
      params: {
          window: traffic.window,
          node_id: traffic.node,
          exclude_nic: traffic.exclude,
          start_time: traffic.window === 'custom' ? traffic.timeRange?.[0] : '',
          end_time: traffic.window === 'custom' ? traffic.timeRange?.[1] : ''
      }
  }).then(res => {
      myChart.hideLoading()
      if ((res.code === 0 || res.code === 200) && res.data) {
          updateChartOption(res.data)
      }
  }).catch(() => {
      myChart.hideLoading()
  })
}

const updateChartOption = (data) => {
    const series = []
    if (traffic.out) {
        series.push({
            name: '出站流量',
            type: 'line',
            data: data.out_traffic || [],
            smooth: true,
            areaStyle: { opacity: 0.1 },
            itemStyle: { color: '#409eff' }
        })
    }
    if (traffic.in) {
        series.push({
             name: '入站流量',
             type: 'line',
             data: data.in_traffic || [],
             smooth: true,
             areaStyle: { opacity: 0.1 },
             itemStyle: { color: '#67c23a' }
        })
    }

    const option = {
        tooltip: { trigger: 'axis' },
        legend: { data: ['出站流量', '入站流量'] },
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: data.x_axis || []
        },
        yAxis: {
            type: 'value',
            name: '流量 (MB)'
        },
        series: series
    }
    myChart.setOption(option, true)
}

watch(activeTab, (val) => {
  if (val === 'traffic') {
    nextTick(() => {
      initChart()
    })
    return
  }
  if (val === 'ranking') {
    refreshRanking()
    return
  }
  if (val === 'metrics') {
    refreshMetrics()
  }
})

// Deep watch for traffic options to validation or reload
watch(() => [traffic.out, traffic.in], () => {
    if (activeTab.value === 'traffic') {
         refreshTraffic()
    }
})

watch(() => [ranking.metric, ranking.window], () => {
  if (activeTab.value === 'ranking') {
    refreshRanking()
  }
})

watch(() => [metrics.metric, metrics.window], ([, window], [, prevWindow]) => {
  if (window !== prevWindow && window !== 'custom') {
    metrics.timeRange = []
  }
  if (activeTab.value === 'metrics') {
    refreshMetrics()
  }
})

watch(() => metrics.timeRange, () => {
  if (activeTab.value !== 'metrics') return
  if (metrics.window !== 'custom') return
  if (metrics.timeRange?.[0] && metrics.timeRange?.[1]) {
    refreshMetrics()
  }
}, { deep: true })

watch(() => traffic.window, (val, prev) => {
  if (val === prev) return
  if (val !== 'custom') {
    traffic.timeRange = []
    if (activeTab.value === 'traffic') refreshTraffic()
    return
  }
  // custom: wait for user to pick a full time range or click refresh.
})

const handleTrafficTimeRangeChange = () => {
  if (traffic.window !== 'custom') return
  if (traffic.timeRange?.[0] && traffic.timeRange?.[1] && activeTab.value === 'traffic') {
    refreshTraffic()
  }
}

</script>

<style scoped>
.monitor-tabs {
  margin-bottom: 12px;
}

.monitor-toolbar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
}

.toolbar-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.toolbar-label {
  color: #606266;
  min-width: 40px;
}

.time-range {
  width: 320px;
}

.chart-container {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 18px;
  background: #fff;
}

.chart-title {
  font-weight: 600;
  margin-bottom: 12px;
}

.traffic-chart {
    height: 400px;
    width: 100%;
}

.refresh-button {
  min-width: 96px;
  width: 96px;
  align-self: flex-start;
}
</style>

