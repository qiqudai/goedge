<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" class="site-tabs">
      <el-tab-pane label="版本管理" name="versions" />
      <el-tab-pane label="同步升级" name="sync" />
    </el-tabs>

    <div v-if="activeTab === 'versions'">
      <el-card class="box-card">
        <template #header>
          <div class="card-header">
            <span>边缘节点版本管理</span>
            <el-button style="float: right; padding: 3px 0" text @click="fetchVersions">刷新</el-button>
          </div>
        </template>

        <div class="upload-row">
          <el-input v-model="uploadVersion" placeholder="版本号，如 1.2.3" style="width: 240px;" />
          <el-upload
            class="upload-demo"
            :action="uploadAction"
            :headers="uploadHeaders"
            :data="{ version: uploadVersion }"
            :limit="1"
            :before-upload="beforeUpload"
            :on-success="handleUploadSuccess"
            accept=".tar.gz,.zip"
          >
            <el-button type="primary">上传新版本 (tar.gz/zip)</el-button>
          </el-upload>
          <div class="upload-tip">文件名将自动包含版本号，上传后写入数据库。</div>
        </div>

        <el-table :data="versionList" style="width: 100%">
          <el-table-column prop="version" label="版本号" width="180" />
          <el-table-column prop="status" label="状态" width="140">
            <template #default="{ row }">
              <el-tag :type="row.status === 'stable' ? 'success' : (row.status === 'gray' ? 'warning' : 'info')">
                {{ row.status === 'stable' ? '稳定版' : row.status || '普通' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="gray_percent" label="灰度比例 %" width="200">
            <template #default="{ row }">
              <div v-if="row.status === 'gray'">
                <el-slider v-model="row.gray_percent" :step="10" show-input @change="handleGrayChange(row)" />
              </div>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="upload_time" label="上传时间" width="200">
            <template #default="{ row }">
              {{ formatTime(row.upload_time) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220">
            <template #default="{ row }">
              <el-button size="small" type="success" v-if="row.status !== 'stable'" @click="promoteToStable(row)">设为稳定版</el-button>
              <el-button size="small" @click="openSyncTab(row)">同步此版本</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <div v-else>
      <el-card class="box-card">
        <template #header>
          <div class="card-header">
            <span>同步升级</span>
            <div class="header-right">
              <el-select v-model="selectedVersion" placeholder="选择版本" style="width: 240px;" @change="fetchNodes">
                <el-option v-for="item in versionList" :key="item.version" :label="item.version" :value="item.version" />
              </el-select>
              <el-button type="primary" :disabled="!selectedVersion" @click="fetchNodes">刷新节点</el-button>
            </div>
          </div>
        </template>

        <div class="sync-toolbar">
          <div class="region-select">
            <span class="label">区域分组:</span>
            <el-checkbox-group v-model="selectedRegions" @change="handleRegionSelect">
              <el-checkbox v-for="group in regionGroups" :key="group.id" :label="group.id">{{ group.name }}</el-checkbox>
            </el-checkbox-group>
          </div>
          <el-button type="primary" :disabled="!selectedVersion || !selectedRows.length" @click="confirmUpgrade">批量升级</el-button>
        </div>

        <el-table
          ref="nodeTable"
          :data="nodeRows"
          border
          fit
          highlight-current-row
          row-key="id"
          :loading="nodeLoading"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="55" align="center" />
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="ip" label="IP" width="140" />
          <el-table-column prop="region_name" label="区域" min-width="140" />
          <el-table-column prop="group_name" label="所属分组" min-width="160" />
          <el-table-column prop="current_version" label="当前版本" width="140" />
          <el-table-column prop="latest_version" label="最新版本" width="140" />
          <el-table-column prop="status" label="状态" width="140">
            <template #default="{ row }">
              <el-tag v-if="row.upgrade_state === 'running'" type="warning">升级中 {{ row.progress }}%</el-tag>
              <el-tag v-else-if="row.upgrade_state === 'success'" type="success">已完成</el-tag>
              <el-tag v-else-if="row.upgrade_state === 'failed_final'" type="danger">升级失败</el-tag>
              <el-tag v-else-if="row.status === 'upgrade_available'" type="warning">待升级</el-tag>
              <el-tag v-else-if="row.status === 'up_to_date'" type="success">已最新</el-tag>
              <el-tag v-else type="info">空闲</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="showLog(row)">日志</el-button>
              <el-button link type="primary" size="small" @click="upgradeSingle(row)">升级</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, onBeforeUnmount } from 'vue'
import request, { API_BASE } from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('versions')
const versionList = ref([])
const uploadVersion = ref('')
const uploadAction = `${API_BASE}/api/v1/admin/packages`
const uploadHeaders = {
  Authorization: 'Bearer ' + localStorage.getItem('admin_token')
}

const selectedVersion = ref('')
const nodeList = ref([])
const nodeLoading = ref(false)
const selectedRows = ref([])
const selectedRegions = ref([])
const upgradeTaskId = ref(0)
const upgradeStatusMap = ref({})
const nodeTable = ref(null)
let pollTimer = null

const fetchVersions = () => {
  request.get('/packages').then(res => {
    versionList.value = res.data?.list || []
  })
}

const beforeUpload = () => {
  if (!uploadVersion.value) {
    ElMessage.warning('请先输入版本号')
    return false
  }
  return true
}

const handleUploadSuccess = () => {
  ElMessage.success('上传成功')
  fetchVersions()
}

const handleGrayChange = (row) => {
  request.post('/packages/grayscale', { version: row.version, percent: row.gray_percent })
    .then(() => ElMessage.success('灰度比例已更新'))
}

const promoteToStable = (row) => {
  request.post('/packages/stable', { version: row.version })
    .then(() => {
      ElMessage.success(`已设为稳定版: ${row.version}`)
      fetchVersions()
    })
}

const openSyncTab = (row) => {
  activeTab.value = 'sync'
  selectedVersion.value = row.version
  fetchNodes()
}

const fetchNodes = () => {
  if (!selectedVersion.value) return
  nodeLoading.value = true
  request.get('/packages/nodes', { params: { version: selectedVersion.value } }).then(res => {
    nodeList.value = res.data?.list || []
    nodeLoading.value = false
    selectedRows.value = []
    selectedRegions.value = []
    upgradeStatusMap.value = {}
  }).catch(() => {
    nodeLoading.value = false
  })
}

const regionGroups = computed(() => {
  const map = new Map()
  nodeList.value.forEach(item => {
    const id = item.region_id || 0
    const name = item.region_name || '未分配'
    if (!map.has(id)) {
      map.set(id, { id, name, nodeIds: [] })
    }
    map.get(id).nodeIds.push(item.id)
  })
  return Array.from(map.values())
})

const handleRegionSelect = () => {
  if (!nodeTable.value) return
  nodeTable.value.clearSelection()
  const regionSet = new Set(selectedRegions.value)
  nodeRows.value.forEach(row => {
    const key = row.region_id || 0
    if (regionSet.has(key)) {
      nodeTable.value.toggleRowSelection(row, true)
    }
  })
}

const handleSelectionChange = rows => {
  selectedRows.value = rows
}

const confirmUpgrade = () => {
  ElMessageBox.confirm(`确认升级选中节点到版本 ${selectedVersion.value} ?`, '提示', {
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    startUpgrade(selectedRows.value.map(row => row.id))
  })
}

const upgradeSingle = (row) => {
  if (!selectedVersion.value) {
    ElMessage.warning('请选择版本')
    return
  }
  startUpgrade([row.id])
}

const startUpgrade = (nodeIds) => {
  if (!nodeIds.length) return
  request.post('/packages/upgrade', { version: selectedVersion.value, node_ids: nodeIds }).then(res => {
    upgradeTaskId.value = res.data?.task_id || 0
    if (upgradeTaskId.value) {
      startPolling()
    }
    ElMessage.success('升级任务已提交')
  })
}

const startPolling = () => {
  stopPolling()
  pollTimer = setInterval(fetchUpgradeStatus, 3000)
}

const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const fetchUpgradeStatus = () => {
  if (!upgradeTaskId.value) return
  request.get('/packages/upgrade/status', { params: { task_id: upgradeTaskId.value } }).then(res => {
    const nodes = res.data?.nodes || []
    const map = {}
    nodes.forEach(item => {
      map[item.node_id] = item
    })
    upgradeStatusMap.value = map
  })
}

const nodeRows = computed(() => {
  const rows = nodeList.value.map(row => {
    const status = upgradeStatusMap.value[row.id] || {}
    return {
      ...row,
      upgrade_state: status.state || '',
      progress: status.progress || 0,
      message: status.message || '',
      upgrade_ret: status.ret || ''
    }
  })
  rows.sort((a, b) => {
    const ra = a.region_name || '未分配'
    const rb = b.region_name || '未分配'
    if (ra !== rb) return ra.localeCompare(rb)
    const ga = a.group_name || ''
    const gb = b.group_name || ''
    if (ga !== gb) return ga.localeCompare(gb)
    const na = a.name || ''
    const nb = b.name || ''
    return na.localeCompare(nb)
  })
  return rows
})

const showLog = (row) => {
  const ret = row.upgrade_ret || row.message
  if (!ret) {
    ElMessage.info('暂无日志')
    return
  }
  ElMessageBox.alert(ret, '升级日志', {
    confirmButtonText: '关闭',
    type: 'info',
    customClass: 'error-dialog-pre'
  })
}

const formatTime = (t) => {
  if (!t) return '-'
  if (typeof t === 'string') {
    return t.replace('T', ' ').substring(0, 19)
  }
  return String(t)
}

onMounted(() => {
  fetchVersions()
})

onBeforeUnmount(() => {
  stopPolling()
})
</script>

<style scoped>
.upload-row {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.upload-tip {
  color: #999;
  font-size: 12px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-right {
  display: flex;
  gap: 8px;
  align-items: center;
}
.sync-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 12px;
}
.region-select {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.region-select .label {
  color: #666;
  font-size: 12px;
}
:deep(.error-dialog-pre .el-message-box__message) { white-space: pre-wrap; word-break: break-all; max-height: 400px; overflow-y: auto; }
</style>
