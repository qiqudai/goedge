<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" @tab-click="handleTabClick">
      
      <!-- 基础数据 -->
      <el-tab-pane label="基础数据" name="basic">
        <div class="chart-wrapper">
          <div ref="bandwidthChartRef" class="chart" />
        </div>
        <div class="chart-wrapper">
          <div ref="trafficChartRef" class="chart" />
        </div>
        <div class="chart-wrapper">
          <div ref="qpsChartRef" class="chart" />
        </div>
      </el-tab-pane>

      <!-- 质量监控 -->
      <el-tab-pane label="质量监控" name="quality">
        <div class="chart-wrapper">
          <div ref="hitRateChartRef" class="chart" />
        </div>
        <div class="chart-wrapper">
          <div ref="statusChartRef" class="chart" />
        </div>
      </el-tab-pane>

      <!-- 回源监控 -->
      <el-tab-pane label="回源监控" name="origin">
        <div class="chart-wrapper">
          <div ref="originBandwidthChartRef" class="chart" />
        </div>
        <div class="chart-wrapper">
          <div ref="originTrafficChartRef" class="chart" />
        </div>
      </el-tab-pane>

      <!-- 数据排行 -->
      <el-tab-pane label="数据排行" name="ranking">
        <div class="filter-container">
           <el-radio-group v-model="rankingType" style="margin-bottom: 20px;" @change="fetchRankingList">
             <el-radio-button value="domain">域名排行</el-radio-button>
             <el-radio-button value="url">热门URL</el-radio-button>
             <el-radio-button value="ip">Top客户端IP</el-radio-button>
             <el-radio-button value="country">国家排行</el-radio-button>
             <el-radio-button value="province">省份排行</el-radio-button>
             <el-radio-button value="referer">热门Referer</el-radio-button>
           </el-radio-group>

           <div style="margin-bottom: 20px;">
                <el-radio-group v-model="timeRange" size="small" @change="fetchRankingList" style="margin-right: 10px;">
                    <el-radio-button value="10min">10分钟实时</el-radio-button>
                    <el-radio-button value="30min">近30分钟</el-radio-button>
                    <el-radio-button value="1h">近1小时</el-radio-button>
                    <el-radio-button value="custom">自定义</el-radio-button>
                </el-radio-group>
                <el-input v-model="rankingKeyword" :placeholder="searchPlaceholder" style="width: 200px;" class="filter-item" @keyup.enter="fetchRankingList" />
                <el-button class="filter-item" type="primary" :icon="Search" @click="fetchRankingList" style="margin-left: 10px;">刷新</el-button>
           </div>
        </div>

        <el-table :data="rankingList" border style="width: 100%" v-loading="loading">
          <el-table-column prop="rank" label="排行" width="80" />
          
          <el-table-column :label="itemLabel" min-width="200">
             <template #default="scope">
                {{ scope.row.item }}
             </template>
          </el-table-column>

          <el-table-column prop="request_count" label="请求次数" sortable />
          <el-table-column prop="out_traffic" label="出站流量" sortable />
          <el-table-column prop="origin_traffic" label="回源流量" sortable />
        </el-table>
      </el-tab-pane>

    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, reactive, watch, nextTick } from 'vue'
import { Search } from '@element-plus/icons-vue'
import request from '@/utils/request'
import * as echarts from 'echarts'

const activeTab = ref('basic') // Default changed to basic to show charts first
const loading = ref(false)

// --- Charts Refs ---
const bandwidthChartRef = ref(null)
const trafficChartRef = ref(null)
const qpsChartRef = ref(null)
const hitRateChartRef = ref(null)
const statusChartRef = ref(null)
const originBandwidthChartRef = ref(null)
const originTrafficChartRef = ref(null)

let charts = [] // Store chart instances to dispose/resize

// --- Ranking ---
const rankingType = ref('domain')
const timeRange = ref('10min')
const rankingKeyword = ref('')
const rankingList = ref([])

const itemLabel = computed(() => {
    const map = {
        'domain': '域名',
        'url': 'URL',
        'ip': 'IP地址',
        'country': '国家',
        'province': '省份',
        'referer': 'Referer'
    }
    return map[rankingType.value] || '项目'
})

const searchPlaceholder = computed(() => {
     const map = {
        'domain': '输入域名',
        'url': '输入URL',
        'ip': '输入IP',
        'country': '输入国家',
        'province': '输入省份',
        'referer': '输入Referer'
    }
    return map[rankingType.value] || '输入关键词'
})


const fetchRankingList = async () => {
  loading.value = true
  try {
    const res = await request.get('/stats/ranking', {
      params: { 
        type: rankingType.value,
        time_range: timeRange.value,
        keyword: rankingKeyword.value
      }
    })
    if (res.code === 0) {
        rankingList.value = res.data.list
    }
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

// --- Chart Data Fetching & Rendering ---
const initChart = (dom, title, xAxisData, seriesData, unit) => {
  if (!dom) return
  const chart = echarts.init(dom)
  chart.setOption({
    title: { text: title },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: xAxisData },
    yAxis: { type: 'value', name: unit },
    series: seriesData
  })
  charts.push(chart)
}

const fetchBasicStats = async () => {
    try {
        const res = await request.get('/stats/basic')
        if (res.code === 0) {
            const data = res.data
            await nextTick()
            initChart(bandwidthChartRef.value, '带宽', data.x_axis, [{ name: 'Bandwidth', type: 'line', data: data.bandwidth, areaStyle: {} }], 'Mbps')
            initChart(trafficChartRef.value, '流量', data.x_axis, [{ name: 'Traffic', type: 'line', data: data.traffic, areaStyle: {} }], 'MB')
            initChart(qpsChartRef.value, 'QPS', data.x_axis, [{ name: 'QPS', type: 'line', data: data.qps, areaStyle: {} }], 'req/s')
        }
    } catch (e) { console.error(e) }
}

const fetchQualityStats = async () => {
    try {
        const res = await request.get('/stats/quality')
        if (res.code === 0) {
            const data = res.data
             await nextTick()
            initChart(hitRateChartRef.value, '请求命中率', data.x_axis, [{ name: 'Hit Rate', type: 'line', data: data.hit_rate }], '%')
            initChart(statusChartRef.value, '状态码', data.x_axis, [
                { name: '4xx', type: 'line', data: data.status_4xx },
                { name: '5xx', type: 'line', data: data.status_5xx }
            ], 'Count')
        }
    } catch (e) { console.error(e) }
}

const fetchOriginStats = async () => {
     try {
        const res = await request.get('/stats/origin')
        if (res.code === 0) {
            const data = res.data
             await nextTick()
            initChart(originBandwidthChartRef.value, '回源带宽', data.x_axis, [{ name: 'Origin Bandwidth', type: 'line', data: data.origin_bandwidth }], 'Mbps')
            initChart(originTrafficChartRef.value, '回源流量', data.x_axis, [{ name: 'Origin Traffic', type: 'line', data: data.origin_traffic }], 'MB')
        }
    } catch (e) { console.error(e) }
}

const handleTabClick = (tab) => {
    // Clear previous charts to reuse DOM properly or just to be safe
    charts.forEach(c => c.dispose())
    charts = []

    if (tab.props.name === 'ranking') {
        fetchRankingList()
    } else if (tab.props.name === 'basic') {
        fetchBasicStats()
    } else if (tab.props.name === 'quality') {
        fetchQualityStats()
    } else if (tab.props.name === 'origin') {
        fetchOriginStats()
    }
}

onMounted(() => {
  // Default load basic
  fetchBasicStats()
})
</script>

<style scoped>
.filter-container {
  margin-bottom: 20px;
}
.filter-item {
  margin-right: 10px;
}
.chart-wrapper {
  margin-bottom: 30px;
}
.chart {
    width: 100%;
    height: 350px;
}
</style>
