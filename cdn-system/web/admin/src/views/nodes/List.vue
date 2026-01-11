<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="pageTab" class="custom-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="节点列表" name="list">
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
            @row-action="handleRowAction"
          />
        </el-tab-pane>

        <el-tab-pane label="区域管理" name="region">
          <RegionList v-if="pageTab === 'region'" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <NodeEditDialog
      v-model="editVisible"
      :item="currentItem"
      @success="fetchList"
    />

    <MonitorLogDialog
      v-model="monitorVisible"
      :node-id="currentNodeId"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

import NodeTable from './list/NodeTable.vue'
import RegionList from './list/RegionList.vue'
import NodeEditDialog from './list/NodeEditDialog.vue'
import MonitorLogDialog from './list/MonitorLogDialog.vue'

const router = useRouter()
const pageTab = ref('list')
const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const selectedRows = ref([])
const regions = ref([])

const editVisible = ref(false)
const currentItem = ref({})

const monitorVisible = ref(false)
const currentNodeId = ref(0)

const fetchList = async (query = {}) => {
  listLoading.value = true
  try {
    const res = await request.get('/nodes', { params: query })
    const rows = res.data?.list || []
    list.value = rows.map((row) => {
      if (!row.enable) {
        return { ...row, status_text: '禁用', status_class: 'disabled' }
      }
      if (row.online) {
        return { ...row, status_text: '在线', status_class: 'online' }
      }
      return { ...row, status_text: '离线', status_class: 'offline' }
    })
    total.value = res.data?.total || 0
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

const handleSearch = (q) => fetchList(q)
const handleCreate = () => { currentItem.value = { id: 0 }; editVisible.value = true }
const handleEdit = (row) => { currentItem.value = { ...row }; editVisible.value = true }

const handleBatch = async (action, ids) => {
  const targetIds = ids || selectedRows.value.map(r => r.id)
  await ElMessageBox.confirm(`确定执行批量${action}操作吗？`, '提示')
  await request.post('/nodes/batch_action', { action, ids: targetIds })
  ElMessage.success('操作成功')
  fetchList()
}

const handleGoGroups = (row) => router.push({ path: '/nodes/groups', query: { node_id: row.id } })
const handleMonitorLogs = (row) => { currentNodeId.value = row.id; monitorVisible.value = true }
const handleGoMonitor = (row) => router.push({ path: '/nodes/monitor', query: { node_id: row.id } })

const handleStatusChange = async (row) => {
  await request.put(`/nodes/${row.id}/status`, { enable: row.enable })
  ElMessage.success('状态更新成功')
}

const handleRowAction = (command, row) => {
  if (command === 'delete') {
    ElMessageBox.confirm('确定删除节点吗？', '提示').then(async () => {
      await request.delete(`/nodes/${row.id}`)
      fetchList()
    })
  }
}

onMounted(() => {
  fetchList()
  fetchRegions()
})
</script>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
</style>
