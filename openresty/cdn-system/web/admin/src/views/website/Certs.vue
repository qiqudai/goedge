<template>
  <div class="app-container">
    <el-tabs v-model="activeTopTab" class="site-tabs" @tab-click="handleTopTab">
      <el-tab-pane label="证书列表" name="list" />
      <el-tab-pane label="默认设置" name="default" />
      <el-tab-pane label="DNS API" name="dns" />
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
          <el-option label="All" value="all" />
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
      @selection-change="handleSelectionChange"
      v-model:current-page="listQuery.page"
      v-model:page-size="listQuery.pageSize"

      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="handleFilter"
      @current-change="handleFilter"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column prop="domain" label="域名" min-width="200" show-overflow-tooltip />
      <el-table-column prop="create_at" label="创建时间" width="180" />
      <el-table-column prop="expire_time" label="到期时间" width="180" />
      <el-table-column label="自动续签" width="100" align="center">
        <template #default="{ row }">
          <el-icon v-if="row.auto_renew" color="#67C23A"><CircleCheckFilled /></el-icon>
          <el-icon v-else color="#C0C4CC"><CircleCloseFilled /></el-icon>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.enable ? 'success' : 'info'">{{ row.enable ? '正常' : '禁用' }}</el-tag>
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
                <el-dropdown-item @click="handleRowAction('delete', row)">删除</el-dropdown-item>
                <el-dropdown-item @click="handleRowAction('force_disable', row)">强制禁用</el-dropdown-item>
                <el-dropdown-item @click="handleDownload(row)">下载</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </AppTable>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="640px">
      <el-tabs v-model="dialogTab" type="card">
        <el-tab-pane label="单个" name="single">
          <el-form :model="form" label-width="90px">
            <el-form-item label="名称">
              <el-input v-model="form.name" placeholder="输入证书名称" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.des" placeholder="备注" />
            </el-form-item>
            <el-form-item label="类型">
              <el-radio-group v-model="form.type">
                <el-radio value="upload">自己上传</el-radio>
                <el-radio value="zerossl">ZeroSSL(推荐)</el-radio>
                <el-radio value="letsencrypt">Let's Encrypt</el-radio>
                <el-radio value="buypass">BuyPass</el-radio>
                <el-radio value="google">Google CA</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.type === 'upload'" label="证书">
              <el-input v-model="form.cert" type="textarea" :rows="4" placeholder="-----BEGIN CERTIFICATE-----" />
            </el-form-item>
            <el-form-item v-if="form.type === 'upload'" label="密钥">
              <el-input v-model="form.key" type="textarea" :rows="4" placeholder="-----BEGIN PRIVATE KEY-----" />
            </el-form-item>
            <el-form-item v-if="form.type !== 'upload'" label="域名">
              <el-input v-model="form.domain" placeholder="输入域名, 多个域名空格分隔" />
            </el-form-item>
            <el-form-item v-if="form.type !== 'upload'" label="DNS API">
              <el-select v-model="form.dnsapi" clearable placeholder="请选择" style="width: 100%;">
              <el-option v-for="d in dnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
              </el-select>
              <div class="help-text">
                这里的 DNS API 仅用于证书申请（DNS 验证），与站点 CNAME 解析无关。
              </div>
            </el-form-item>
            <el-form-item label="自动续签">
              <el-switch v-model="form.auto_renew" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="批量申请" name="batch">
          <el-form :model="batchForm" label-width="90px">
            <el-form-item label="类型">
              <el-radio-group v-model="batchForm.type">
                <el-radio value="zerossl">ZeroSSL(推荐)</el-radio>
                <el-radio value="letsencrypt">Let's Encrypt</el-radio>
                <el-radio value="buypass">BuyPass</el-radio>
                <el-radio value="google">Google CA</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="域名">
              <el-input v-model="batchForm.domains" type="textarea" :rows="5" placeholder="输入域名，一行一个" />
            </el-form-item>
            <el-form-item label="DNS API">
              <el-select v-model="batchForm.dnsapi" clearable placeholder="请选择" style="width: 100%;">
              <el-option v-for="d in dnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
              </el-select>
              <div class="help-text">
                这里的 DNS API 仅用于证书申请（DNS 验证），与站点 CNAME 解析无关。
              </div>
            </el-form-item>
            <el-form-item label="自动续签">
              <el-switch v-model="batchForm.auto_renew" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

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
              <el-radio value="zerossl">ZeroSSL(推荐)</el-radio>
              <el-radio value="letsencrypt">Let's Encrypt</el-radio>
              <el-radio value="buypass">BuyPass</el-radio>
              <el-radio value="google">Google CA</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="DNS API">
            <el-select v-model="defaultForm.dnsapi" clearable placeholder="请选择" style="width: 320px;">
              <el-option v-for="d in defaultDnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
            <div class="help-text">
              这里的 DNS API 仅用于证书申请（DNS 验证），与站点 CNAME 解析无关。
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="saveDefaultSettings">保存</el-button>
          </el-form-item>
        </template>

        <div v-else class="default-empty">请先选择用户</div>
      </el-form>
    </el-card>

    <div v-if="activeTopTab === 'dns'" class="dnsapi-section">
      <div class="filter-container">
        <el-button type="primary" @click="openDnsapiDialog">新增DNS API</el-button>
        <el-button :disabled="!selectedDnsapi.length" @click="removeDnsapiBatch">删除</el-button>
      </div>
      <el-table v-loading="dnsapiLoading" :data="dnsapiList" border style="width: 100%;" @selection-change="handleDnsapiSelection">
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column prop="type" label="DNS" width="140">
          <template #default="{ row }">
            <el-tag>{{ formatDnsType(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="140" align="center">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openDnsapiEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="removeDnsapi(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="dnsapiDialogVisible" title="新增DNS API" width="520px">
      <el-form :model="dnsapiForm" label-width="90px">
        <el-form-item label="名称" required>
          <el-input v-model="dnsapiForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="dnsapiForm.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item label="DNS" required>
          <el-select v-model="dnsapiForm.type" placeholder="请选择" style="width: 100%;" @change="resetDnsapiAuth">
            <el-option v-for="t in dnsapiTypes" :key="t.type" :label="t.name" :value="t.type" />
          </el-select>
        </el-form-item>
        <el-form-item label="验证信息" v-if="currentDnsapiType">
          <div class="dnsapi-fields">
            <el-form-item
              v-for="field in currentDnsapiType.fields"
              :key="field"
              :label="dnsapiFieldLabel(dnsapiForm.type, field)"
              label-width="120px"
            >
              <el-input v-model="dnsapiForm.credentials[field]" />
            </el-form-item>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dnsapiDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitDnsapi">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import request from '@/utils/request'
const activeTopTab = ref('list')
const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const selectedRows = ref([])
const dnsapiOptions = ref([])
const dnsapiList = ref([])
const dnsapiLoading = ref(false)
const selectedDnsapi = ref([])
const dnsapiTypes = ref([])
const dnsapiDialogVisible = ref(false)
const isAdmin = ref((localStorage.getItem('role') || 'user') === 'admin')
const userOptions = ref([])
const userLoading = ref(false)
const selectedDefaultUser = ref(0)
const adminDefaultDnsapiOptions = ref([])
const dnsapiForm = reactive({
  id: 0,
  name: '',
  remark: '',
  type: '',
  credentials: {}
})
const defaultForm = reactive({
  type: 'system',
  dnsapi: ''
})
const defaultDnsapiOptions = computed(() => (
  isAdmin.value ? adminDefaultDnsapiOptions.value : dnsapiOptions.value
))

const listQuery = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  searchField: 'domain'
})

const dialogVisible = ref(false)
const dialogTab = ref('single')
const editingId = ref(0)
const form = reactive({
  name: '',
  des: '',
  type: 'upload',
  domain: '',
  dnsapi: '',
  cert: '',
  key: '',
  auto_renew: false
})
const batchForm = reactive({
  type: 'zerossl',
  domains: '',
  dnsapi: '',
  auto_renew: true
})

const dialogTitle = computed(() => (editingId.value ? '编辑证书' : '添加证书'))

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

const handleSelectionChange = rows => {
  selectedRows.value = rows
}

const openCreate = () => {
  editingId.value = 0
  dialogTab.value = 'single'
  form.name = ''
  form.des = ''
  form.type = 'upload'
  form.domain = ''
  form.dnsapi = ''
  form.cert = ''
  form.key = ''
  form.auto_renew = false
  batchForm.type = 'zerossl'
  batchForm.domains = ''
  batchForm.dnsapi = ''
  batchForm.auto_renew = true
  dialogVisible.value = true
}

const openEdit = row => {
  editingId.value = row.id
  dialogTab.value = 'single'
  form.name = row.name || ''
  form.des = row.des || ''
  form.type = row.type || 'upload'
  form.domain = row.domain || ''
  form.dnsapi = row.dnsapi || ''
  form.cert = row.cert || ''
  form.key = row.key || ''
  form.auto_renew = !!row.auto_renew
  dialogVisible.value = true
}

const submitForm = () => {
  if (dialogTab.value === 'batch') {
    request.post('/certs/batch', batchForm).then(res => {
      ElMessage.success(res.message || '批量申请提交成功')
      dialogVisible.value = false
      fetchList()
    })
    return
  }

  const payload = { ...form }
  if (editingId.value) {
    request.put(`/certs/${editingId.value}`, payload).then(() => {
      ElMessage.success('更新成功')
      dialogVisible.value = false
      fetchList()
    })
  } else {
    request.post('/certs', payload).then(() => {
      ElMessage.success('添加成功')
      dialogVisible.value = false
      fetchList()
    })
  }
}

const handleBatchAction = action => {
  if (!selectedRows.value.length) {
    ElMessage.warning('请选择证书')
    return
  }
  const ids = selectedRows.value.map(row => row.id)
  ElMessageBox.confirm('确定执行该操作?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.post('/certs/batch_action', { action, ids }).then(res => {
      ElMessage.success(res.message || '操作成功')
      fetchList()
    })
  })
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
  request.get(`/certs/${row.id}/download`, { responseType: 'blob' }).then(res => {
    const blob = new Blob([res], { type: 'application/octet-stream' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${row.name || 'cert'}-${row.id}.pem`
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
  dnsapiLoading.value = true
  request.get('/dnsapi').then(res => {
    dnsapiList.value = res.data?.list || res.list || []
    dnsapiOptions.value = dnsapiList.value
    dnsapiLoading.value = false
  }).catch(() => {
    dnsapiLoading.value = false
  })
}

const loadDnsapiTypes = () => {
  request.get('/dnsapi/types').then(res => {
    dnsapiTypes.value = res.data?.types || res.list || []
  })
}

const currentDnsapiType = computed(() => dnsapiTypes.value.find(t => t.type === dnsapiForm.type))

const dnsapiFieldLabel = (type, field) => {
  const mapping = {
    aliyun: { id: 'AccessKey ID', secret: 'AccessKey Secret' },
    huawei: { id: 'Access Key ID', secret: 'Secret Access Key' },
    dnsla: { id: 'API ID', secret: 'API密钥' },
    dnspod: { id: 'ID', token: 'Token' },
    cloudflare: { email: 'Email', key: 'API Key' },
    godaddy: { key: 'Key', secret: 'Secret' }
  }
  if (mapping[type] && mapping[type][field]) {
    return mapping[type][field]
  }
  return field.toUpperCase()
}

const formatDnsType = type => {
  const t = dnsapiTypes.value.find(item => item.type === type)
  return t ? t.name : type
}

const resetDnsapiAuth = () => {
  dnsapiForm.credentials = {}
}

const openDnsapiDialog = () => {
  dnsapiForm.id = 0
  dnsapiForm.name = ''
  dnsapiForm.remark = ''
  dnsapiForm.type = ''
  dnsapiForm.credentials = {}
  dnsapiDialogVisible.value = true
}

const openDnsapiEdit = row => {
  dnsapiForm.id = row.id
  dnsapiForm.name = row.name
  dnsapiForm.remark = row.remark || ''
  dnsapiForm.type = row.type
  dnsapiForm.credentials = row.auth ? JSON.parse(row.auth) : {}
  dnsapiDialogVisible.value = true
}

const submitDnsapi = () => {
  if (!dnsapiForm.name || !dnsapiForm.type) {
    ElMessage.warning('请填写完整信息')
    return
  }
  const payload = {
    name: dnsapiForm.name,
    remark: dnsapiForm.remark,
    type: dnsapiForm.type,
    auth: JSON.stringify(dnsapiForm.credentials || {})
  }
  if (dnsapiForm.id) {
    request.put(`/dnsapi/${dnsapiForm.id}`, payload).then(() => {
      ElMessage.success('更新成功')
      dnsapiDialogVisible.value = false
      loadDnsapiList()
    })
  } else {
    request.post('/dnsapi', payload).then(() => {
      ElMessage.success('创建成功')
      dnsapiDialogVisible.value = false
      loadDnsapiList()
    })
  }
}

const removeDnsapi = row => {
  ElMessageBox.confirm('确认删除该DNS API?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.delete(`/dnsapi/${row.id}`).then(() => {
      ElMessage.success('删除成功')
      loadDnsapiList()
    })
  })
}

const handleDnsapiSelection = rows => {
  selectedDnsapi.value = rows
}

const removeDnsapiBatch = () => {
  if (!selectedDnsapi.value.length) return
  const ids = selectedDnsapi.value.map(row => row.id)
  Promise.all(ids.map(id => request.delete(`/dnsapi/${id}`))).then(() => {
    ElMessage.success('删除成功')
    loadDnsapiList()
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

const loadDefaultDnsapiList = (userId) => {
  if (!userId) {
    adminDefaultDnsapiOptions.value = []
    return
  }
  request.get('/dnsapi', { params: { user_id: userId } }).then(res => {
    adminDefaultDnsapiOptions.value = res.data?.list || res.list || []
  })
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
  loadDefaultDnsapiList(userId)
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
  loadDnsapiTypes()
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
.default-empty {
  color: #909399;
  font-size: 12px;
  padding: 4px 0 4px 90px;
}
</style>

