<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="pageTab" class="custom-tabs" @tab-change="handleTabChange">
        <el-tab-pane :label="NODE_T.nodeListTab" name="list">
          <NodeTable
            :list="list"
            :total="total"
            :loading="listLoading"
            :selected-rows="selectedRows"
            :regions="regions"
            @search="handleSearch"
            @create="handleCreate"
            @edit="handleEdit"
            @batch="handleBatch"
            @refresh="fetchList"
            @selection-change="(rows) => selectedRows = rows"
            @go-groups="handleGoGroups"
            @monitor-logs="handleMonitorLogs"
            @go-monitor="handleGoMonitor"
            @status-change="handleStatusChange"
            @anti-blocking-change="handleAntiBlockingChange"
            @row-action="handleRowAction"
          />
        </el-tab-pane>

        <el-tab-pane :label="NODE_T.regionManageTab" name="region">
          <RegionList v-if="pageTab === 'region'" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <NodeEditDialog
      v-model="editVisible"
      :item="currentItem"
      :regions="regions"
      @success="fetchList"
    />

    <MonitorLogDialog
      v-model="monitorVisible"
      :node-id="currentNodeId"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

import NodeTable from './list/NodeTable.vue'
import RegionList from './list/RegionList.vue'
import NodeEditDialog from './list/NodeEditDialog.vue'
import MonitorLogDialog from './list/MonitorLogDialog.vue'
import { NODE_T } from './list/constants'

const INSTALL_TIMEOUT = 10000

const INSTALL_POLL_INTERVAL = 5000

const router = useRouter()
const pageTab = ref('list')
const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const selectedRows = ref([])
const regions = ref([])
const currentQuery = ref({})

const editVisible = ref(false)
const currentItem = ref({})

const monitorVisible = ref(false)
const currentNodeId = ref(0)
let installPollingTimer = null

const applyNodeStatus = (row) => {
  const antiBlocking = row.anti_blocking !== false
  if (!row.enable) {
    return { ...row, anti_blocking: antiBlocking, status_text: '\u7981\u7528', status_class: 'disabled' }
  }
  if (row.online) {
    return { ...row, anti_blocking: antiBlocking, status_text: '\u5728\u7ebf', status_class: 'online' }
  }
  return { ...row, anti_blocking: antiBlocking, status_text: '\u79bb\u7ebf', status_class: 'offline' }
}

const setNodeStatus = (row) => {
  if (!row) return
  const next = applyNodeStatus(row)
  row.status_text = next.status_text
  row.status_class = next.status_class
}

const startInstallPolling = () => {
  if (installPollingTimer) return
  installPollingTimer = setInterval(() => {
    if (!listLoading.value) fetchList()
  }, INSTALL_POLL_INTERVAL)
}

const stopInstallPolling = () => {
  if (!installPollingTimer) return
  clearInterval(installPollingTimer)
  installPollingTimer = null
}

const updateInstallPolling = (rows) => {
  const hasRunning = rows.some((row) => String(row.install_status || '').toLowerCase() === 'running')
  if (hasRunning) {
    startInstallPolling()
  } else {
    stopInstallPolling()
  }
}

const fetchList = async (query = currentQuery.value) => {
  const nextQuery = { ...query }
  currentQuery.value = nextQuery
  listLoading.value = true
  try {
    const res = await request.get('/nodes', { params: nextQuery })
    const rows = res.data?.list || []
    list.value = rows.map((row) => applyNodeStatus(row))
    total.value = res.data?.total || 0
    updateInstallPolling(list.value)
  } finally {
    listLoading.value = false
  }
}

const fetchRegions = async () => {
  const res = await request.get('/regions')
  regions.value = res.data?.list || []
}

const handleTabChange = (name) => {
  if (name === 'list') fetchList()
}

const handleSearch = (q) => fetchList({ ...q })
const handleCreate = () => { currentItem.value = { id: 0 }; editVisible.value = true }
const handleEdit = (row) => { currentItem.value = { ...row }; editVisible.value = true }

const handleBatch = async (action, ids) => {
  if (action === 'delete') {
    const idSet = new Set((ids || selectedRows.value.map(r => r.id)))
    const rows = list.value.filter(row => idSet.has(row.id))
    const hasBindings = rows.some(row => Number(row.line_count || 0) > 0)
    if (hasBindings) {
      ElMessage.warning('该节点已加入线路分组，请先移除后再删除')
      return
    }
  }
  const targetIds = ids || selectedRows.value.map(r => r.id)
  await ElMessageBox.confirm(`\u786e\u5b9a\u6267\u884c\u6279\u91cf${action}\u64cd\u4f5c\u5417\uff1f`, '\u63d0\u793a')
  await request.post('/nodes/batch_action', { action, ids: targetIds })
  ElMessage.success('\u64cd\u4f5c\u6210\u529f')
  fetchList()
}

const handleGoGroups = (row) => router.push({ path: '/nodes/groups', query: { node_id: row.id } })
const handleMonitorLogs = (row) => { currentNodeId.value = row.id; monitorVisible.value = true }
const handleGoMonitor = (row) => router.push({ path: '/nodes/monitor', query: { node_id: row.id } })

const handleStatusChange = async (row) => {
  const targetEnable = row.enable
  try {
    await request.put(`/nodes/${row.id}/status`, { enable: row.enable })
    setNodeStatus(row)
    ElMessage.success('\u72b6\u6001\u66f4\u65b0\u6210\u529f')
  } catch (err) {
    row.enable = !targetEnable
    setNodeStatus(row)
    ElMessage.error('\u72b6\u6001\u66f4\u65b0\u5931\u8d25')
  }
}

const handleAntiBlockingChange = async (row) => {
  const targetEnable = row.anti_blocking
  try {
    await request.put(`/nodes/${row.id}/anti_blocking`, { enable: row.anti_blocking })
    ElMessage.success('防屏蔽状态更新成功，已下发到节点')
  } catch (err) {
    row.anti_blocking = !targetEnable
    ElMessage.error('防屏蔽状态更新失败')
  }
}

const handleRowAction = (command, row) => {
  if (command === 'delete') {
    if (Number(row.line_count || 0) > 0) {
      ElMessage.warning('该节点已加入线路分组，请先移除后再删除')
      return
    }
    ElMessageBox.confirm('\u786e\u5b9a\u5220\u9664\u8282\u70b9\u5417\uff1f', '\u63d0\u793a').then(async () => {
      await request.delete(`/nodes/${row.id}`)
      fetchList()
    })
    return
  }
  if (command === 'install') {
    ElMessageBox.confirm('\u786e\u5b9a\u91cd\u65b0\u5b89\u88c5\u8282\u70b9\u5417\uff1f', '\u63d0\u793a').then(async () => {
      const res = await request.post(`/nodes/${row.id}/install`, {}, { timeout: INSTALL_TIMEOUT })
      ElMessage.success('\u64cd\u4f5c\u6210\u529f')
      if (res?.install_error) {
        ElMessage.warning(`\u5b89\u88c5\u5931\u8d25: ${res.install_error}`)
      }
      fetchList()
    })
  }
}

onMounted(() => {
  fetchList()
  fetchRegions()
})

onBeforeUnmount(() => {
  stopInstallPolling()
})
</script>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
</style>

