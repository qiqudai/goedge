<template>
  <div class="app-container">
    <el-tabs v-model="activeTopTab" class="site-tabs" @tab-click="handleTopTab">
      <el-tab-pane label="证书列表" name="list" />
      <el-tab-pane label="默认设置" name="default" />
      <el-tab-pane label="DNS 接口" name="dns" />
    </el-tabs>

    <div v-if="activeTopTab === 'list'" class="filter-container">
      <div class="filter-left">
        <el-button type="primary" @click="openCreate">添加证书</el-button>
        <el-button :disabled="!selectedRows.length" @click="handleReissue">重新申请</el-button>
        <el-dropdown trigger="click">
          <el-button>
            更多操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleBatchAction('enable')">启用</el-dropdown-item>
                <el-dropdown-item @click="handleBatchAction('disable')">禁用</el-dropdown-item>
                <el-dropdown-item @click="handleBatchAction('auto_renew_enable')">开启续签</el-dropdown-item>
                <el-dropdown-item @click="handleBatchAction('auto_renew_disable')">关闭续签</el-dropdown-item>
                <el-dropdown-item @click="handleBatchAction('delete')">删除</el-dropdown-item>
                <el-dropdown-item @click="handleBatchAction('force_disable')">强制禁用</el-dropdown-item>
                <el-dropdown-item @click="handleDownloadBatch">下载</el-dropdown-item>
              </el-dropdown-menu>
            </template>
        </el-dropdown>
      </div>

      <div class="filter-right">
        <el-select v-model="listQuery.searchField" class="filter-item" style="width: 120px;">
          <el-option label="名称" value="name" />
          <el-option label="域名" value="domain" />
          <el-option label="类型" value="type" />
          <el-option label="全部" value="all" />
        </el-select>
        <el-input
          v-model="listQuery.keyword"
          placeholder="输入名称/域名, 模糊搜索"
          style="width: 260px;"
          class="filter-item"
          @keyup.enter="handleFilter"
        />
        <el-button type="primary" class="filter-item" @click="handleFilter">搜索</el-button>
      </div>
    </div>

    <AppTable
      v-if="activeTopTab === 'list'"
      :loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      persist-key="website-certs-list"
      storage-key="website-certs-list"
      @selection-change="handleSelectionChange"
      v-model:current-page="listQuery.page"
      v-model:page-size="listQuery.pageSize"

      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="handleSizeChange"
      @current-change="handlePageChange"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column v-if="isAdmin" prop="user" label="用户" width="120">
        <template #default="{row}">
          <span v-if="row.user_name">{{ formatUserLabel({id: row.uid, name: row.user_name}) }}</span>
          <span v-else>uid: {{ row.uid }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="名称" width="320" show-overflow-tooltip>
        <template #default="{row}">
          <span class="link-type" @click="openEdit(row)">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="160" />
      <el-table-column prop="domain" label="域名" width="240" show-overflow-tooltip>
        <template #default="{row}">
           <span>{{ row.domain }}</span>
           <el-icon class="copy-icon" @click.stop="copyText(row.domain)" v-if="row.domain"><CopyDocument /></el-icon>
        </template>
      </el-table-column>
      <el-table-column prop="create_at" label="创建时间" width="200">
        <template #default="{row}">
           {{ formatTime(row.create_at) }}
        </template>
      </el-table-column>
      <el-table-column prop="expire_time" label="到期时间" width="200">
        <template #default="{row}">
           {{ formatTime(row.expire_time) }}
        </template>
      </el-table-column>
      <el-table-column label="自动续签" width="90" align="center">
        <template #default="{ row }">
          <el-icon v-if="row.auto_renew" color="#67C23A"><CircleCheckFilled /></el-icon>
          <el-icon v-else color="#C0C4CC"><CircleCloseFilled /></el-icon>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-tag v-if="!row.enable" type="danger" size="small">禁用</el-tag>
          <el-tag v-else-if="row.state === 'dns_pending'" type="warning" size="small">DNS验证中</el-tag>
          <el-tag v-else-if="row.state === 'waiting'" type="info" size="small">待签发</el-tag>
          <el-tag v-else-if="row.state === 'issuing'" type="warning" size="small">签发中</el-tag>
          <el-tag v-else-if="row.state === 'ready' || row.state === 'success' || !row.state" type="success" size="small">已签发</el-tag>
          <el-tag v-else-if="row.state === 'fail'" type="danger" size="small">失败</el-tag>
          <el-tag v-else type="info" size="small">正常</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="失败原因" min-width="150" show-overflow-tooltip>
         <template #default="{ row }">
                                <span
                                  v-if="row.state === 'fail' && (row.ret || row.issue_task_ret)"
                                  class="error-text"
                                  @click="showError(row.ret || row.issue_task_ret)"
                                >
                                   {{ row.ret || row.issue_task_ret }}
                                </span>
            <span v-else>-</span>
         </template>
      </el-table-column>
      <el-table-column label="操作" width="150" align="center">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openEdit(row)">管理</el-button>
          <el-dropdown trigger="click">
            <span class="link-more">
              更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleRowAction('enable', row)">启用</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('disable', row)">禁用</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('auto_renew_enable', row)">开启续签</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('auto_renew_disable', row)">关闭续签</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('delete', row)">删除</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('force_disable', row)">强制禁用</el-dropdown-item>
                <el-dropdown-item @click="handleDownload(row)">下载</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </AppTable>

    <CertEditPopup
      v-model:visible="popupVisible"
      :certId="editingId"
      :isAdmin="isAdmin"
      :initialData="currentCertData"
      @saved="fetchList"
    />

    <el-card v-if="activeTopTab === 'default'" class="default-card">
      <el-form :model="defaultForm" label-width="90px">
        <el-form-item v-if="isAdmin" label="用户">
          <el-select
            v-model="selectedDefaultUser"
            filterable
            remote
            clearable
            placeholder="输入ID、邮箱、用户名、手机号搜索"
            :remote-method="loadUsers"
            :loading="userLoading"
            @change="handleDefaultUserChange">
            <el-option v-for="u in userOptions" :key="u.id" :label="formatUserLabel(u)" :value="u.id" />
          </el-select>
        </el-form-item>

        <template v-if="!isAdmin || selectedDefaultUser">
          <el-form-item label="证书类型">
            <el-radio-group v-model="defaultForm.type">
              <el-radio value="system">系统默认设置</el-radio>
              <el-radio value="zerossl">ZeroSSL（推荐）</el-radio>
              <el-radio value="letsencrypt">Let's Encrypt</el-radio>
              <el-radio value="buypass">BuyPass</el-radio>
              <el-radio value="google">Google CA</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="DNS 接口">
            <el-select v-model="defaultForm.dnsapi" clearable placeholder="请选择" style="width: 320px;">
              <el-option v-for="d in defaultDnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
            <div class="help-text">
              这里的 DNS 接口仅用于证书申请（DNS 验证），与站点 CNAME 解析无关。
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="saveDefaultSettings">保存</el-button>
          </el-form-item>
        </template>

        <div v-else class="default-empty">请先选择用户</div>
      </el-form>
    </el-card>

    <DnsApiTab
      v-if="activeTopTab === 'dns'"
      @list-updated="handleDnsapiListUpdated"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, CircleCheckFilled, CircleCloseFilled, CopyDocument } from '@element-plus/icons-vue'
import request from '@/utils/request'
import CertEditPopup from './CertEditPopup.vue'
import DnsApiTab from './components/DnsApiTab.vue'

const activeTopTab = ref('list')
const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const selectedRows = ref([])
const dnsapiOptions = ref([])
const isAdmin = ref((localStorage.getItem('role') || 'user') === 'admin')
const userOptions = ref([])
const userLoading = ref(false)
const selectedDefaultUser = ref(0)
const defaultForm = reactive({
  type: 'system',
  dnsapi: ''
})
const defaultDnsapiOptions = computed(() => dnsapiOptions.value)

const listQuery = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  searchField: 'domain'
})


const popupVisible = ref(false)
const editingId = ref(0)
const currentCertData = ref({})

const handleTopTab = () => {}

const fetchList = () => {
  listLoading.value = true
  request.get('/certs', {
    params: {
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      keyword: listQuery.keyword,
      search_field: listQuery.searchField
    }
  }).then(res => {
    list.value = res.list || res.data || []
    total.value = res.total || 0
    listLoading.value = false
  }).catch(() => {
    listLoading.value = false
  })
}

const handleFilter = () => {
  listQuery.page = 1
  fetchList()
}

const handleSizeChange = (size) => {
  listQuery.pageSize = Number(size) || listQuery.pageSize
  listQuery.page = 1
  fetchList()
}

const handlePageChange = (page) => {
  listQuery.page = Number(page) || 1
  fetchList()
}

const handleSelectionChange = rows => {
  selectedRows.value = rows
}

const formatTime = (t) => {
  if (!t) return '-'
  // Format: YYYY-MM-DD HH:mm:ss
  // Assuming backend returns RFC3339 string. 
  // If strict +08 is needed, might need date library, but simple replacement is standard in this project.
  return t.replace('T', ' ').substring(0, 19)
}

const showError = (err) => {
  ElMessageBox.alert(err, '错误详情', {
    confirmButtonText: '关闭',
    type: 'error',
    customClass: 'error-dialog-pre'
  })
}

const openCreate = () => {
  editingId.value = 0
  currentCertData.value = {}
  popupVisible.value = true
}

const openEdit = row => {
  editingId.value = row.id
  currentCertData.value = { ...row }
  popupVisible.value = true
}

const copyText = (text) => {
  if (!text) return
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const handleBatchAction = action => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择证书')
    return
  }
  const ids = selectedRows.value.map(row => row.id)
  
  const execute = () => {
    request.post('/certs/batch_action', { action, ids }).then(res => {
      ElMessage.success(res.message || '操作成功')
      fetchList()
    })
  }

  if (action === 'delete') {
    ElMessageBox.confirm('确定删除选中证书?', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }).then(() => {
      execute()
    })
  } else {
    execute()
  }
}

const handleRowAction = (action, row) => {
  selectedRows.value = [row]
  handleBatchAction(action)
}

const handleReissue = () => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择证书')
    return
  }
  const ids = selectedRows.value.map(row => row.id)
  request.post('/certs/reissue', { ids }).then(res => {
    ElMessage.success(res.message || '已提交重新申请')
    fetchList()
  })
}

const handleDownload = row => {
  request.get(`/certs/${row.id}/download`, { responseType: 'blob', params: { domain: row.domain } }).then(res => {
    const blob = new Blob([res], { type: 'application/octet-stream' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${row.name || 'cert'}-${row.id}.zip`
    link.click()
    window.URL.revokeObjectURL(url)
  })
}

const handleDownloadBatch = () => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择证书')
    return
  }
  selectedRows.value.forEach(row => handleDownload(row))
}

const loadDnsapiList = () => {
  request.get('/dnsapi').then(res => {
    dnsapiOptions.value = res.data?.list || res.list || []
  }).catch(() => {
    dnsapiOptions.value = []
  })
}

const formatUserLabel = user => {
  if (!user) return ''
  const name = user.username || user.name || ''
  return name ? `${name} (id: ${user.id})` : `id: ${user.id}`
}

const loadUsers = query => {
  if (query !== '') {
    userLoading.value = true
    request.get('/users', { params: { keyword: query, pageSize: 20 } }).then(res => {
      userOptions.value = res.data?.list || res.list || []
      userLoading.value = false
    }).catch(() => {
      userLoading.value = false
    })
  }
}

const loadDefaultSettings = (userId) => {
  const params = userId ? { user_id: userId } : undefined
  request.get('/certs/default_settings', { params }).then(res => {
    const data = res.data || {}
    defaultForm.type = data.type || 'system'
    defaultForm.dnsapi = data.dnsapi || ''
  }).catch(() => {
    defaultForm.type = 'system'
    defaultForm.dnsapi = ''
  })
}

const handleDefaultUserChange = (userId) => {
  defaultForm.type = 'system'
  defaultForm.dnsapi = ''
  loadDefaultSettings(userId)
}

const handleDnsapiListUpdated = (list) => {
  dnsapiOptions.value = Array.isArray(list) ? list : []
}


const saveDefaultSettings = () => {
  if (isAdmin.value && !selectedDefaultUser.value) {
    ElMessage.warning('请选择用户')
    return
  }
  const payload = { ...defaultForm }
  if (isAdmin.value) {
    payload.user_id = selectedDefaultUser.value
  }
  request.post('/certs/default_settings', payload).then(() => {
    ElMessage.success('保存成功')
  })
}

onMounted(() => {
  fetchList()
  loadDnsapiList()
  if (!isAdmin.value) {
    loadDefaultSettings()
  }
})
</script>

<style scoped>
.filter-container {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.filter-left,
.filter-right {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.pagination-container {
  margin-top: 16px;
  text-align: right;
}
.help-text {
  font-size: 12px;
  color: #909399;
  margin-top: 6px;
}
.link-more {
  color: #409eff;
  cursor: pointer;
  font-size: 12px;
  margin-left: 8px;
}
.error-text { color: #f56c6c; cursor: pointer; text-decoration: underline; }
.error-text:hover { color: #f78989; }
:deep(.error-dialog-pre .el-message-box__message) { white-space: pre-wrap; word-break: break-all; max-height: 400px; overflow-y: auto; }
.default-empty {
  color: #909399;
  font-size: 12px;
  padding: 4px 0 4px 90px;
}
.link-type {
  color: #409eff;
  cursor: pointer;
}
.link-type:hover {
  text-decoration: underline;
}
.copy-icon {
  margin-left: 5px;
  cursor: pointer;
  color: #909399;
  vertical-align: middle;
}
.copy-icon:hover {
  color: #409eff;
}
</style>
