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
          <el-radio-group v-model="ranking.metric" size="normal">
            <el-radio-button label="bandwidth">带宽</el-radio-button>
            <el-radio-button label="connection">连接</el-radio-button>
            <el-radio-button label="load">负载</el-radio-button>
            <el-radio-button label="disk">硬盘</el-radio-button>
          </el-radio-group>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">时间</span>
          <el-radio-group v-model="ranking.window" size="normal">
            <el-radio-button label="1m">1分钟</el-radio-button>
            <el-radio-button label="5m">5分钟</el-radio-button>
            <el-radio-button label="30m">30分钟</el-radio-button>
            <el-radio-button label="1h">1小时</el-radio-button>
          </el-radio-group>
        </div>
        <el-button type="primary" size="normal" class="refresh-button" style="width: 96px;" @click="refreshRanking">刷新</el-button>
      </div>

      <AppTable :data="ranking.list" border persist-key="node-ranking">
        <el-table-column prop="rank" label="排行" width="80" align="center" />
        <el-table-column prop="node" label="节点" min-width="160" />
        <el-table-column prop="nic" label="网卡" min-width="120" />
        <el-table-column prop="out" label="出站带宽" min-width="140" />
        <el-table-column prop="in" label="入站带宽" min-width="140" />
      </AppTable>
    </div>

    <div v-else-if="activeTab === 'metrics'">
      <div class="monitor-toolbar">
        <div class="toolbar-row">
          <span class="toolbar-label">指标</span>
          <el-radio-group v-model="metrics.metric" size="normal">
            <el-radio-button label="bandwidth">带宽</el-radio-button>
            <el-radio-button label="connection">连接</el-radio-button>
            <el-radio-button label="load">负载</el-radio-button>
            <el-radio-button label="disk">硬盘</el-radio-button>
          </el-radio-group>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">时间</span>
          <el-radio-group v-model="metrics.window" size="normal">
            <el-radio-button label="1h">1小时</el-radio-button>
            <el-radio-button label="6h">6小时</el-radio-button>
            <el-radio-button label="12h">12小时</el-radio-button>
          </el-radio-group>
          <el-date-picker
            v-model="metrics.timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            class="time-range"
            style="width: 320px;"
          />
        </div>
        <el-button type="primary" size="normal" class="refresh-button" style="width: 96px;" @click="refreshMetrics">刷新</el-button>
      </div>

      <AppTable :data="metrics.list" border persist-key="node-metrics">
        <el-table-column prop="time" label="时间" min-width="160" />
        <el-table-column prop="value" label="数值" min-width="120" />
      </AppTable>
    </div>

    <div v-else>
      <div class="monitor-toolbar">
        <div class="toolbar-row">
          <span class="toolbar-label">类型</span>
          <el-checkbox v-model="traffic.out">出站流量</el-checkbox>
          <el-checkbox v-model="traffic.in">入站流量</el-checkbox>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">时间</span>
          <el-radio-group v-model="traffic.window" size="normal">
            <el-radio-button label="1d">1天</el-radio-button>
            <el-radio-button label="7d">7天</el-radio-button>
            <el-radio-button label="30d">30天</el-radio-button>
            <el-radio-button label="custom">自定义</el-radio-button>
          </el-radio-group>
          <el-date-picker
            v-model="traffic.timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            clearable
            class="time-range"
            style="width: 320px;"
          />
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">节点</span>
          <el-select v-model="traffic.node" style="width: 220px;">
            <el-option v-for="item in nodeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </div>
        <div class="toolbar-row">
          <span class="toolbar-label">排除网卡</span>
          <el-input v-model="traffic.exclude" placeholder="排除网卡，多个网卡用空格分隔" style="width: 320px;" />
        </div>
        <el-button type="primary" size="normal" class="refresh-button" style="width: 96px;" @click="refreshTraffic">刷新</el-button>
      </div>

      <div class="chart-placeholder">
        <div class="chart-title">节点流量</div>
        <div class="chart-empty">暂无数据</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'

const activeTab = ref('ranking')

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

const nodeOptions = [
  { label: 'agent', value: 'agent' },
  { label: '全部节点', value: 'all' }
]

const refreshRanking = () => {
  ranking.list = []
}

const refreshMetrics = () => {
  metrics.list = []
}

const refreshTraffic = () => {
  // Placeholder for future API integration.
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

.chart-placeholder {
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 18px;
  min-height: 240px;
}

.chart-title {
  font-weight: 600;
  margin-bottom: 12px;
}

.chart-empty {
  color: #909399;
}

.refresh-button {
  min-width: 96px;
  width: 96px;
  align-self: flex-start;
}
</style>
