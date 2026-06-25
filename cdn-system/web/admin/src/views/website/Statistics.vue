<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" @tab-click="handleTabClick">
      
      <!-- 基础数据 -->
      <el-tab-pane label="基础数据" name="basic" lazy>
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
      <el-tab-pane label="质量监控" name="quality" lazy>
        <div class="chart-wrapper">
          <div ref="hitRateChartRef" class="chart" />
        </div>
        <div class="chart-wrapper">
          <div ref="statusChartRef" class="chart" />
        </div>
      </el-tab-pane>

      <!-- 回源监控 -->
      <el-tab-pane label="回源监控" name="origin" lazy>
        <div class="chart-wrapper">
          <div ref="originBandwidthChartRef" class="chart" />
        </div>
        <div class="chart-wrapper">
          <div ref="originTrafficChartRef" class="chart" />
        </div>
      </el-tab-pane>

      <!-- 数据排行 -->
      <el-tab-pane label="数据排行" name="ranking" lazy>
        <div class="filter-container">
           <el-radio-group v-model="rankingType" style="margin-bottom: 20px;" @change="handleRankingTypeChange">
             <el-radio-button value="domain">域名排行</el-radio-button>
             <el-radio-button value="url">热门URL</el-radio-button>
             <el-radio-button value="url_ip">热门URL及IP</el-radio-button>
             <el-radio-button value="latency">耗时排行</el-radio-button>
             <el-radio-button value="ip">客户端IP排行</el-radio-button>
             <el-radio-button value="country">国家排行</el-radio-button>
             <el-radio-button value="province">省份排行</el-radio-button>
             <el-radio-button value="referer">热门来源</el-radio-button>
           </el-radio-group>

           <div style="margin-bottom: 20px;">
                <el-radio-group v-model="timeRange" size="small" @change="handleTimeRangeChange" style="margin-right: 10px;">
                    <el-radio-button value="10min">10分钟实时</el-radio-button>
                    <el-radio-button value="30min">30分钟</el-radio-button>
                    <el-radio-button value="1h">1小时</el-radio-button>
                    <el-radio-button value="custom">自定义</el-radio-button>
                </el-radio-group>
                <el-date-picker
                  v-if="timeRange === 'custom'"
                  v-model="customTimeRange"
                  type="datetimerange"
                  value-format="YYYY-MM-DD HH:mm:ss"
                  start-placeholder="开始时间"
                  end-placeholder="结束时间"
                  range-separator="至"
                  style="width: 380px; margin-right: 10px;"
                  @change="handleCustomRangeChange"
                />
                <el-input v-model="rankingKeyword" :placeholder="searchPlaceholder" style="width: 200px;" class="filter-item" @keyup.enter="handleRankingSearch" />
                <el-button class="filter-item" type="primary" :icon="Search" @click="handleRankingSearch" style="margin-left: 10px;">刷新</el-button>
           </div>
        </div>

        <el-table :data="rankingList" border style="width: 100%" v-loading="loading">
          <el-table-column v-if="isHotURLIP" type="expand" width="48">
            <template #default="scope">
              <el-table
                :data="scope.row.ips || []"
                border
                size="small"
                class="ip-detail-table"
                empty-text="暂无IP访问"
              >
                <el-table-column prop="rank" label="排行" width="80" />
                <el-table-column prop="ip" label="IP地址" min-width="180" />
                <el-table-column prop="total_request_count" label="统计范围请求次数" sortable />
                <el-table-column prop="request_count" label="60秒访问次数" sortable />
                <el-table-column label="操作" width="120" align="center">
                  <template #default="ipScope">
                    <el-button
                      size="small"
                      type="danger"
                      link
                      @click="blockHotURLIP(scope.row, ipScope.row)"
                    >
                      加入黑名单
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </template>
          </el-table-column>
          <el-table-column prop="rank" label="排行" width="80" />
          
          <el-table-column :label="itemLabel" min-width="200">
             <template #default="scope">
                {{ scope.row.item }}
             </template>
          </el-table-column>

        <el-table-column v-if="!isLatency" prop="request_count" label="请求次数" sortable />
        <el-table-column v-if="!isLatency" prop="out_traffic" label="出站流量" sortable />
        <el-table-column v-if="!isLatency" prop="origin_traffic" label="回源流量" sortable />
        <el-table-column v-if="isLatency" prop="avg_time" label="平均耗时(ms)" sortable>
          <template #default="scope">
            <span class="latency-chip latency-avg">{{ formatMilliseconds(scope.row.avg_time) }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="isLatency" prop="max_time" label="最大耗时(ms)" sortable>
          <template #default="scope">
            <span class="latency-chip latency-max">{{ formatMilliseconds(scope.row.max_time) }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="isLatency" prop="min_time" label="最小耗时(ms)" sortable>
          <template #default="scope">
            <span class="latency-chip latency-min">{{ formatMilliseconds(scope.row.min_time) }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="isLatency" prop="p95_time" label="P95耗时(ms)" sortable>
          <template #default="scope">
            {{ formatMilliseconds(scope.row.p95_time) }}
          </template>
        </el-table-column>
        <el-table-column v-if="isLatency" prop="request_count" label="请求次数" sortable />
      </el-table>
      <div class="pagination-container">
        <AppPagination
          v-model:current-page="rankingPager.page"
          v-model:page-size="rankingPager.pageSize"
          :total="rankingPager.total"
          persist-key="website-ranking"
          @current-change="fetchRankingList"
          @size-change="handleRankingPageSizeChange"
        />
      </div>
      </el-tab-pane>

    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, reactive, watch, nextTick } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'
import { loadEcharts } from '@/utils/echarts'
import AppPagination from '@/components/AppPagination.vue'

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
let echarts = null

// --- Ranking ---
const rankingType = ref('domain')
const timeRange = ref('10min')
const rankingKeyword = ref('')
const rankingList = ref([])
const customTimeRange = ref([])
const rankingPager = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const itemLabel = computed(() => {
    const map = {
        'domain': '域名',
        'url': 'URL',
        'url_ip': 'URL',
        'latency': 'URL',
        'ip': 'IP地址',
        'country': '国家',
        'province': '省份',
        'referer': '来源'
    }
    return map[rankingType.value] || '项目'
})

const searchPlaceholder = computed(() => {
     const map = {
        'domain': '输入域名',
        'url': '输入URL',
        'url_ip': '输入URL或域名',
        'latency': '输入URL',
        'ip': '输入IP',
        'country': '输入国家',
        'province': '输入省份',
        'referer': '输入来源'
    }
    return map[rankingType.value] || 'Enter keyword'
})

const isLatency = computed(() => rankingType.value === 'latency')
const isHotURLIP = computed(() => rankingType.value === 'url_ip')

const formatMilliseconds = (value) => {
  if (value === null || value === undefined || Number.isNaN(Number(value))) {
    return '-'
  }
  return Number(value).toFixed(2)
}


const fetchRankingList = async () => {
  if (timeRange.value === 'custom') {
    const validCustomRange = Array.isArray(customTimeRange.value)
      && customTimeRange.value.length === 2
      && customTimeRange.value[0]
      && customTimeRange.value[1]
    if (!validCustomRange) {
      ElMessage.warning('请选择完整的自定义时间范围')
      return
    }
  }
  loading.value = true
  try {
    const params = {
      type: rankingType.value,
      time_range: timeRange.value,
      keyword: rankingKeyword.value,
      page: rankingPager.page,
      pageSize: rankingPager.pageSize
    }
    if (timeRange.value === 'custom' && Array.isArray(customTimeRange.value) && customTimeRange.value.length === 2) {
      params.start_time = customTimeRange.value[0]
      params.end_time = customTimeRange.value[1]
    }
    const res = await request.get('/stats/ranking', {
      params
    })
    if (res.code === 0 || res.code === 200) {
        rankingList.value = res.data.list || []
        rankingPager.total = Number(res.data.total) || 0
        rankingPager.page = Number(res.data.page) || rankingPager.page
        rankingPager.pageSize = Number(res.data.pageSize) || rankingPager.pageSize
    }
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

const resetRankingPage = () => {
  rankingPager.page = 1
}

const handleRankingTypeChange = () => {
  resetRankingPage()
  fetchRankingList()
}

const handleRankingSearch = () => {
  resetRankingPage()
  fetchRankingList()
}

const handleCustomRangeChange = (val) => {
  if (timeRange.value !== 'custom') return
  if (Array.isArray(val) && val.length === 2 && val[0] && val[1]) {
    resetRankingPage()
    fetchRankingList()
  }
}

const handleTimeRangeChange = (val) => {
  if (val === 'custom') return
  resetRankingPage()
  fetchRankingList()
}

const handleRankingPageSizeChange = () => {
  rankingPager.page = 1
  fetchRankingList()
}

const blockHotURLIP = async (urlRow, ipRow) => {
  const ip = ipRow?.ip
  const domain = urlRow?.site
  if (!ip || !domain) {
    ElMessage.warning('缺少IP或域名，无法加入黑名单')
    return
  }
  try {
    await ElMessageBox.confirm(`确定将 ${ip} 加入 ${domain} 的IP黑名单？`, '加入黑名单', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })
    const res = await request.post('/logs/block/block_ip', { ip, domain })
    if (res.code === 0 || res.code === 200) {
      ElMessage.success(res.data?.added === false ? '该IP已在黑名单中' : '已加入黑名单并同步配置')
    }
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    console.error(error)
  }
}

// --- Chart Data Fetching & Rendering ---
const initChart = async (dom, title, xAxisData, seriesData, unit) => {
  if (!dom) return
  echarts = echarts || await loadEcharts()
  if (!document.body.contains(dom)) return
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
        if (res.code === 0 || res.code === 200) {
            const data = res.data
            await nextTick()
            await Promise.all([
              initChart(bandwidthChartRef.value, '带宽', data.x_axis, [{ name: '带宽', type: 'line', data: data.bandwidth, areaStyle: {} }], 'Mbps'),
              initChart(trafficChartRef.value, '流量', data.x_axis, [{ name: '流量', type: 'line', data: data.traffic, areaStyle: {} }], 'MB'),
              initChart(qpsChartRef.value, 'QPS', data.x_axis, [{ name: 'QPS', type: 'line', data: data.qps, areaStyle: {} }], 'req/s')
            ])
        }
    } catch (e) { console.error(e) }
}

const fetchQualityStats = async () => {
    try {
        const res = await request.get('/stats/quality')
        if (res.code === 0 || res.code === 200) {
            const data = res.data
             await nextTick()
            await Promise.all([
              initChart(hitRateChartRef.value, 'Hit Rate', data.x_axis, [{ name: 'Hit Rate', type: 'line', data: data.hit_rate }], '%'),
              initChart(statusChartRef.value, '状态码', data.x_axis, [
                  { name: '4xx', type: 'line', data: data.status_4xx },
                  { name: '5xx', type: 'line', data: data.status_5xx }
              ], 'Count')
            ])
        }
    } catch (e) { console.error(e) }
}

const fetchOriginStats = async () => {
     try {
        const res = await request.get('/stats/origin')
        if (res.code === 0 || res.code === 200) {
            const data = res.data
             await nextTick()
            await Promise.all([
              initChart(originBandwidthChartRef.value, '回源带宽', data.x_axis, [{ name: 'Origin Bandwidth', type: 'line', data: data.origin_bandwidth }], 'Mbps'),
              initChart(originTrafficChartRef.value, '回源流量', data.x_axis, [{ name: 'Origin Traffic', type: 'line', data: data.origin_traffic }], 'MB')
            ])
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
.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
.ip-detail-table {
  width: calc(100% - 48px);
  margin: 8px 0 8px 48px;
}
.latency-chip {
  --latency-text: #1f2937;
  --latency-border: rgba(0, 0, 0, 0.08);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 64px;
  padding: 2px 10px;
  border-radius: 999px;
  border: 1px solid var(--latency-border);
  color: var(--latency-text);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.2px;
}
.latency-avg {
  background: linear-gradient(135deg, #e8f4ff, #f7fbff);
}
.latency-max {
  --latency-text: #7a2f00;
  background: linear-gradient(135deg, #ffe8d1, #fff4ea);
  border-color: rgba(255, 132, 31, 0.3);
}
.latency-min {
  --latency-text: #0f5132;
  background: linear-gradient(135deg, #e6f7ef, #f4fbf7);
  border-color: rgba(25, 135, 84, 0.3);
}
</style>

