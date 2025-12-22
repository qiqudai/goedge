<template>
  <div class="dashboard-container">
    <el-row :gutter="16" class="dashboard-row">
      <el-col :span="18">
        <el-card v-if="isAdmin" class="ops-card" shadow="never">
          <template #header>
            <div class="card-header">
              <span>运营数据</span>
              <el-radio-group v-model="opsRange" size="small" class="range-tabs">
                <el-radio-button label="7d">近7日</el-radio-button>
                <el-radio-button label="30d">近30日</el-radio-button>
                <el-radio-button label="last_month">上个月</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <el-row :gutter="12">
            <el-col :span="8">
              <div class="ops-item">
                <div class="ops-title">注册用户</div>
                <div class="ops-value">{{ opsSummary.users || '无数据' }}</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="ops-item">
                <div class="ops-title">开通套餐</div>
                <div class="ops-value">{{ opsSummary.packages || '无数据' }}</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="ops-item">
                <div class="ops-title">充值金额</div>
                <div class="ops-value">{{ opsSummary.recharge || '无数据' }}</div>
              </div>
            </el-col>
          </el-row>
        </el-card>

        <el-card v-else class="user-card" shadow="never">
          <div class="user-card-body">
            <el-avatar :size="48" :src="userInfo.avatar || defaultAvatar" class="user-avatar" />
            <div class="user-content">
              <div class="user-name">
                {{ userInfo.username || '-' }}
                <el-tag size="small" type="info" class="user-tag">{{ userInfo.level || 'V0' }}</el-tag>
              </div>
              <div class="user-meta">
                账号ID：{{ userInfo.id || '-' }}
                <span class="meta-sep">|</span>
                实名认证：{{ userInfo.auth_state || '未认证' }}
              </div>
              <div class="user-meta">
                最后登录：{{ userInfo.last_login || '-' }}
                <span class="meta-sep">|</span>
                登录IP：{{ userInfo.login_ip || '-' }}
              </div>
            </div>
          </div>
        </el-card>

        <el-card class="overview-card" shadow="never">
          <template #header>
            <div class="card-header">
              <span>网络概览</span>
              <el-radio-group v-model="overviewRange" size="small" class="range-tabs">
                <el-radio-button label="today">今日</el-radio-button>
                <el-radio-button label="yesterday">昨日</el-radio-button>
                <el-radio-button label="7d">近7日</el-radio-button>
                <el-radio-button label="30d">近30日</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div class="overview-grid">
            <div class="overview-item">
              <div class="overview-title">带宽峰值</div>
              <div class="overview-value">{{ overview.bandwidth_peak || '-' }}</div>
            </div>
            <div class="overview-item">
              <div class="overview-title">请求数</div>
              <div class="overview-value">{{ overview.requests || '-' }}</div>
            </div>
            <div class="overview-item">
              <div class="overview-title">总流量</div>
              <div class="overview-value">{{ overview.traffic || '-' }}</div>
            </div>
            <div class="overview-item">
              <div class="overview-title">拉黑IP数</div>
              <div class="overview-value">{{ overview.blocked_ips || '-' }}</div>
            </div>
          </div>
        </el-card>

        <el-row :gutter="16" class="panel-row">
          <el-col :span="16">
            <el-card class="chart-card" shadow="never">
              <template #header>
                <div class="card-header">
                  <span>监控趋势</span>
                  <el-radio-group v-model="chartRange" size="small" class="range-tabs" @change="updateChart">
                    <el-radio-button label="today">今日</el-radio-button>
                    <el-radio-button label="yesterday">昨日</el-radio-button>
                    <el-radio-button label="7d">近7日</el-radio-button>
                    <el-radio-button label="30d">近30日</el-radio-button>
                  </el-radio-group>
                </div>
              </template>
              <div class="chart-tabs">
                <el-radio-group v-model="chartType" size="small" @change="updateChart">
                  <el-radio-button label="bandwidth">带宽</el-radio-button>
                  <el-radio-button label="requests">请求数</el-radio-button>
                  <el-radio-button label="traffic">流量</el-radio-button>
                  <el-radio-button label="blocked">拉黑IP数</el-radio-button>
                </el-radio-group>
              </div>
              <div id="trendChart" class="trend-chart"></div>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card class="top-card" shadow="never">
              <template #header>
                <div class="card-header">
                  <span>TOP10 数据(近30分钟)</span>
                </div>
              </template>
              <div class="top-tabs">
                <el-radio-group v-model="topTab" size="small">
                  <el-radio-button label="domain">域名</el-radio-button>
                  <el-radio-button label="url">URL</el-radio-button>
                  <el-radio-button label="ip">IP</el-radio-button>
                  <el-radio-button label="country">国家</el-radio-button>
                </el-radio-group>
              </div>
              <el-table :data="topRows" size="small" :max-height="320" class="top-table">
                <el-table-column prop="name" label="名称" min-width="140" show-overflow-tooltip />
                <el-table-column prop="count" label="次数" width="90" align="right" />
                <el-table-column prop="traffic" label="流量" width="90" align="right" />
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </el-col>

      <el-col :span="6">
        <template v-if="isAdmin">
          <el-card class="admin-card" shadow="never">
            <div class="admin-user">
              <el-avatar :size="48" :src="userInfo.avatar || defaultAvatar" class="user-avatar" />
              <div class="admin-user-info">
                <div class="admin-user-name">{{ userInfo.username || '-' }}</div>
                <div class="admin-user-meta">账号ID：{{ userInfo.id || '-' }}</div>
                <div class="admin-user-meta">最后登录：{{ userInfo.last_login || '-' }}</div>
                <div class="admin-user-meta">登录IP：{{ userInfo.login_ip || '-' }}</div>
              </div>
            </div>
          </el-card>

          <el-card class="admin-card" shadow="never">
            <template #header>
              <div class="card-header">
                <span>系统状态</span>
              </div>
            </template>
            <div class="status-list">
              <div class="status-row">
                <span>主控状态</span>
                <span class="status-right">
                  <span class="status-dot" :class="statusDot(systemStatus.master)"></span>
                </span>
              </div>
              <div class="status-row">
                <span>Elasticsearch</span>
                <span class="status-right">
                  <span class="status-dot" :class="statusDot(systemStatus.elastic)"></span>
                </span>
              </div>
              <div class="status-row">
                <span>Agent状态</span>
                <span class="status-right">
                  <span class="status-dot" :class="statusDot(systemStatus.agent)"></span>
                  <el-button size="small" type="primary" text>立即检查</el-button>
                </span>
              </div>
              <div class="status-tip">Agent状态上次检查时间 {{ systemStatus.checked_at || '-' }}</div>
            </div>
          </el-card>

          <el-card class="admin-card" shadow="never">
            <template #header>
              <div class="card-header">
                <span>系统授权</span>
              </div>
            </template>
            <div class="license-list">
              <div class="status-row">
                <span>授权节点</span>
                <span>{{ licenseInfo.total_nodes || '-' }}</span>
              </div>
              <div class="status-row">
                <span>当前节点</span>
                <span>{{ licenseInfo.current_nodes || '-' }}</span>
              </div>
              <div class="status-row">
                <span>到期时间</span>
                <span>{{ licenseInfo.expire_at || '-' }}</span>
              </div>
              <div class="status-row">
                <span>操作</span>
                <el-button size="small" type="primary">刷新授权</el-button>
              </div>
            </div>
          </el-card>
        </template>

        <template v-else>
          <el-card class="sidebar-card" shadow="never">
            <template #header>
              <div class="card-header">
                <span>系统公告</span>
              </div>
            </template>
            <ul class="announcement-list">
              <li v-for="item in announcements" :key="item.id">
                <span class="title">{{ item.title }}</span>
                <span class="time">{{ item.time }}</span>
              </li>
              <li v-if="!announcements.length" class="empty-row">暂无公告</li>
            </ul>
          </el-card>

          <el-card class="sidebar-card" shadow="never">
            <template #header>
              <div class="card-header">
                <span>套餐流量</span>
              </div>
            </template>
            <div class="package-info">
              <div class="pkg-name">{{ packageInfo.name || '-' }}</div>
              <div class="pkg-desc">{{ packageInfo.desc || '不限, 已用0GB' }}</div>
              <el-progress :percentage="packageInfo.percent || 0" :status="packageInfo.percent > 90 ? 'exception' : ''" />
            </div>
          </el-card>

          <el-card class="sidebar-card" shadow="never">
            <template #header>
              <div class="card-header">
                <span>使用统计</span>
              </div>
            </template>
            <div class="resource-list">
              <div class="status-row">
                <span>域名数</span>
                <span>{{ resources.domains || 0 }}</span>
              </div>
              <div class="status-row">
                <span>转发数</span>
                <span>{{ resources.forward || 0 }}</span>
              </div>
              <div class="status-row">
                <span>证书</span>
                <span>{{ resources.certs || 0 }}</span>
              </div>
              <div class="status-row">
                <span>已购套餐</span>
                <span>{{ resources.packages || 0 }}</span>
              </div>
            </div>
          </el-card>
        </template>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, computed, watch } from 'vue'
import * as echarts from 'echarts'
import request from '@/utils/request'

const defaultAvatar = 'https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png'
const isAdmin = ref((localStorage.getItem('role') || 'user') === 'admin')

const userInfo = ref({})
const overview = ref({})
const chartData = ref({})
const announcements = ref([])
const packageInfo = ref({})
const resources = ref({})
const opsSummary = ref({})
const systemStatus = ref({})
const licenseInfo = ref({})

const chartType = ref('bandwidth')
const chartRange = ref('today')
const overviewRange = ref('today')
const opsRange = ref('7d')
const topTab = ref('domain')
const topLists = ref({
  domain: [],
  url: [],
  ip: [],
  country: []
})

let myChart = null

const statusDot = value => {
  if (value === true || value === 'ok') return 'status-success'
  if (value === false || value === 'fail') return 'status-failed'
  return 'status-default'
}

const normalizeTopList = list => {
  if (!Array.isArray(list)) return []
  return list.map(item => ({
    name: item.name || item.item || item.domain || item.value || '-',
    count: item.count || item.requests || item.hits || '-',
    traffic: item.traffic || item.flow || '-'
  }))
}

const topRows = computed(() => normalizeTopList(topLists.value[topTab.value]))

const initChart = () => {
  const chartDom = document.getElementById('trendChart')
  if (!chartDom) return
  myChart = echarts.init(chartDom)
  updateChartOption()
  window.addEventListener('resize', () => myChart && myChart.resize())
}

const updateChartOption = () => {
  if (!myChart) return
  const xAxis = chartData.value.x_axis || []
  const dataMap = {
    bandwidth: { name: '带宽', data: chartData.value.bandwidth || [], unit: 'Mbps', color: '#3a7bff' },
    requests: { name: '请求数', data: chartData.value.requests || [], unit: '次', color: '#409eff' },
    traffic: { name: '流量', data: chartData.value.traffic || [], unit: 'MB', color: '#67c23a' },
    blocked: { name: '拉黑IP数', data: chartData.value.blocked || [], unit: '个', color: '#e6a23c' }
  }

  const current = dataMap[chartType.value]
  const option = {
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 20, bottom: 30 },
    xAxis: {
      type: 'category',
      data: xAxis,
      boundaryGap: false,
      axisLine: { lineStyle: { color: '#e4e7ed' } },
      axisLabel: { color: '#606266' }
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: '#eef1f6' } },
      axisLabel: { color: '#606266' }
    },
    series: [
      {
        name: current.name,
        type: 'line',
        data: current.data,
        smooth: true,
        symbol: 'circle',
        symbolSize: 4,
        lineStyle: { color: current.color, width: 2 },
        itemStyle: { color: current.color },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64,158,255,0.25)' },
            { offset: 1, color: 'rgba(64,158,255,0.05)' }
          ])
        }
      }
    ]
  }
  myChart.setOption(option, true)
}

const updateChart = () => {
  updateChartOption()
}

const fetchData = async () => {
  try {
    const res = await request.get('/dashboard')
    const data = res.data || {}

    userInfo.value = data.user || {}
    if (userInfo.value.role) {
      isAdmin.value = userInfo.value.role === 'admin'
    }

    const stats = data.stats || {}
    overview.value = {
      bandwidth_peak: stats.bandwidth_peak || stats.bandwidth || '-',
      requests: stats.requests || '-',
      traffic: stats.traffic || '-',
      blocked_ips: stats.blocked_ips || '-'
    }

    chartData.value = data.charts || {}
    topLists.value = {
      domain: data.top_domains || [],
      url: data.top_urls || [],
      ip: data.top_ips || [],
      country: data.top_countries || []
    }

    announcements.value = data.announcements || []
    packageInfo.value = data.package || {}
    resources.value = data.resources || {}
    opsSummary.value = data.ops?.summary || {}
    systemStatus.value = data.system_status || {}
    licenseInfo.value = data.license || {}

    nextTick(() => {
      if (!myChart) initChart()
      updateChartOption()
    })
  } catch (error) {
    console.error('Failed to fetch dashboard data', error)
  }
}

watch(chartType, () => updateChartOption())
watch(chartRange, () => updateChartOption())

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.dashboard-container {
  padding: 16px 20px 24px;
  background: #f5f6f8;
  min-height: calc(100vh - 84px);
}
.dashboard-row {
  align-items: flex-start;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.range-tabs :deep(.el-radio-button__inner) {
  padding: 4px 10px;
}

.user-card {
  margin-bottom: 16px;
}
.user-card-body {
  display: flex;
  align-items: center;
  gap: 16px;
}
.user-avatar {
  background: #e4efff;
}
.user-content {
  flex: 1;
}
.user-name {
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}
.user-tag {
  font-size: 12px;
}
.user-meta {
  color: #606266;
  font-size: 12px;
  margin-top: 6px;
}
.meta-sep {
  margin: 0 6px;
  color: #dcdfe6;
}

.ops-card {
  margin-bottom: 16px;
}
.ops-item {
  border-bottom: 1px solid #ebeef5;
  padding-bottom: 16px;
}
.ops-title {
  color: #909399;
  font-size: 13px;
}
.ops-value {
  font-size: 14px;
  color: #909399;
  margin-top: 30px;
}

.overview-card {
  margin-bottom: 16px;
}
.overview-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
}
.overview-item {
  min-height: 64px;
}
.overview-title {
  font-size: 12px;
  color: #909399;
}
.overview-value {
  font-size: 18px;
  font-weight: 600;
  margin-top: 10px;
}

.panel-row {
  margin-bottom: 16px;
}
.chart-card {
  min-height: 420px;
}
.chart-tabs {
  margin-bottom: 12px;
}
.trend-chart {
  height: 300px;
}

.top-card {
  min-height: 420px;
}
.top-tabs {
  margin-bottom: 12px;
}

.sidebar-card,
.admin-card {
  margin-bottom: 16px;
}

.announcement-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.announcement-list li {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #606266;
  padding: 6px 0;
}
.announcement-list .time {
  color: #909399;
}
.empty-row {
  color: #909399;
}

.package-info {
  font-size: 12px;
}
.pkg-name {
  font-weight: 600;
  margin-bottom: 6px;
}
.pkg-desc {
  color: #909399;
  margin-bottom: 8px;
}

.resource-list .status-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  font-size: 12px;
  color: #606266;
}

.admin-user {
  display: flex;
  align-items: center;
  gap: 12px;
}
.admin-user-info {
  font-size: 12px;
  color: #606266;
}
.admin-user-name {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 6px;
}
.admin-user-meta {
  margin-top: 4px;
}

.status-list {
  font-size: 12px;
  color: #606266;
}
.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
}
.status-right {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.status-success {
  background: #67c23a;
}
.status-failed {
  background: #f56c6c;
}
.status-default {
  background: #c0c4cc;
}
.status-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}

.license-list .status-row {
  font-size: 12px;
  color: #606266;
}
</style>
