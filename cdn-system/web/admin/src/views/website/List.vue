<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="activeTab" class="custom-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="网站列表" name="list">
          <SiteTable
            v-if="activeTab === 'list'"
            :list="siteList"
            :total="totalSites"
            :loading="listLoading"
            :selected-rows="selectedSites"
            :is-admin="isAdmin"
            @search="handleSearch"
            @action="handleSiteAction"
            @selection-change="(rows) => selectedSites = rows"
            @manage="handleManage"
            @export="handleExport"
            @advanced="advancedVisible = true"
          />
        </el-tab-pane>

        <el-tab-pane label="默认设置" name="default">
          <DefaultSettings v-if="activeTab === 'default'" :is-admin="isAdmin" />
        </el-tab-pane>

        <el-tab-pane label="DNS 接口" name="dns">
          <DnsApiTab v-if="activeTab === 'dns'" />
        </el-tab-pane>

        <el-tab-pane label="解析检测" name="resolve">
          <ResolvePage v-if="activeTab === 'resolve'" :hide-tabs="true" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- Global Advanced Search -->
    <el-dialog v-model="advancedVisible" class="site-list-dialog" title="高级搜索" width="500px">
       <el-form label-width="100px">
          <el-form-item label="状态">
             <el-select v-model="advQuery.status" style="width: 100%;">
                <el-option label="正常" value="enabled" />
                <el-option label="停用" value="disabled" />
             </el-select>
          </el-form-item>
       </el-form>
       <template #footer>
          <el-button @click="advancedVisible = false">取消</el-button>
          <el-button type="primary" @click="applyAdvancedSearch">搜索</el-button>
       </template>
    </el-dialog>

    <SiteEditDialog
      v-model="siteEditVisible"
      :data="siteEditData"
      :is-admin="isAdmin"
      @success="fetchSites"
    />

    <BatchEditDialog
      v-model="batchEditVisible"
      :mode="batchEditMode"
      :ids="batchEditIds"
      @success="handleBatchEditSuccess"
    />

    <BatchSettingsDialog
      v-model="batchSettingsVisible"
      :ids="batchSettingsIds"
      @success="fetchSites"
    />

    <TaskMonitorDialog
      v-model="taskMonitorVisible"
      :task-id="taskMonitorId"
      :title="taskMonitorTitle"
      @completed="handleTaskCompleted"
    />

    <el-dialog v-model="resolveCheckVisible" class="site-list-dialog" title="解析检测结果" width="900px" :close-on-click-modal="false">
      <div v-if="resolveCheckLoading" style="color: #909399; margin-bottom: 10px;">正在检测解析，请稍候...</div>
      <el-table :data="resolveCheckResults" size="small" border style="width: 100%" max-height="420">
        <el-table-column prop="domain" label="域名" min-width="200" />
        <el-table-column prop="expectedCname" label="期望CNAME" min-width="220" show-overflow-tooltip />
        <el-table-column prop="resolvedCname" label="解析CNAME" min-width="220" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.ok ? 'success' : 'danger'" size="small">{{ row.ok ? '正常' : '异常' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ips" label="IP" min-width="180" show-overflow-tooltip />
        <el-table-column prop="error" label="错误" min-width="160" show-overflow-tooltip />
      </el-table>
      <template #footer>
        <el-button @click="resolveCheckVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="applyCertVisible"
      class="site-list-dialog apply-cert-dialog"
      title="批量申请证书"
      width="560px"
      :close-on-click-modal="false"
    >
      <div class="apply-cert-summary">
        <div>已选站点：{{ applyCertRows.length }} 个</div>
        <div v-if="applyCertWildcardCount > 0">包含泛域名站点：{{ applyCertWildcardCount }} 个</div>
      </div>
      <el-alert
        v-if="applyCertWildcardCount > 0"
        type="warning"
        :closable="false"
        class="apply-cert-alert"
      >
        已包含泛域名，系统将按 DNS-01 方式申请。
      </el-alert>
      <div class="apply-cert-list">
        <div
          v-for="row in applyCertRows"
          :key="row.id"
          class="apply-cert-item"
        >
          <span class="apply-cert-name">#{{ row.id }}</span>
          <span>{{ row.domain_display || (row.domains && row.domains[0]) || '-' }}</span>
        </div>
      </div>
      <template #footer>
        <el-button @click="applyCertVisible = false">取消</el-button>
        <el-button type="primary" :loading="applyCertSubmitting" @click="submitApplyCert">提交申请</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

import SiteTable from './list/SiteTable.vue'
import SiteEditDialog from './list/SiteEditDialog.vue'
import BatchEditDialog from './list/BatchEditDialog.vue'
import BatchSettingsDialog from './list/BatchSettingsDialog.vue'
import DefaultSettings from './list/DefaultSettings.vue'
import DnsApiTab from './components/DnsApiTab.vue'
import ResolvePage from './Resolve.vue'
import TaskMonitorDialog from '@/components/TaskMonitorDialog.vue'

const router = useRouter()
const isAdmin = ref(localStorage.getItem('role') === 'admin')
const activeTab = ref('list')

// Site List State
const siteList = ref([])
const totalSites = ref(0)
const listLoading = ref(false)
const selectedSites = ref([])
const siteQuery = reactive({ page: 1, pageSize: 10, keyword: '', searchField: 'all' })

const siteEditVisible = ref(false)
const siteEditData = ref(null)
const createDisabled = ref(false)
const createDisabledTip = ref('')

const batchEditVisible = ref(false)
const batchEditMode = ref('')
const batchEditIds = ref([])

const batchSettingsVisible = ref(false)
const batchSettingsIds = ref([])

const taskMonitorVisible = ref(false)
const taskMonitorId = ref('')
const taskMonitorTitle = ref('任务详情')

const resolveCheckVisible = ref(false)
const resolveCheckLoading = ref(false)
const resolveCheckResults = ref([])
const applyCertVisible = ref(false)
const applyCertSubmitting = ref(false)
const applyCertRows = ref([])

const advancedVisible = ref(false)
const advQuery = reactive({ status: '' })

const handleTabChange = (name) => {
  if (name === 'list') {
    fetchSites()
    loadDomainUsage()
  }
}

const fetchSites = async () => {
  listLoading.value = true
  try {
    const res = await request.get('/sites', { params: { ...siteQuery, ...advQuery } })
    siteList.value = res.data?.list || res.list || []
    totalSites.value = res.data?.total || res.total || 0
  } finally {
    listLoading.value = false
  }
}

const handleSearch = (q) => {
  Object.assign(siteQuery, q)
  fetchSites()
}

const handleManage = (row) => {
  router.push({ path: '/website/manage', query: { site_id: row.id } })
}

const handleSiteAction = async (type, data) => {
  if (type === 'create') {
    if (createDisabled.value) {
      ElMessage.error(createDisabledTip.value || '域名数量超限，无法添加')
      return
    }
    siteEditData.value = null
    siteEditVisible.value = true
    return
  }
  if (type === 'edit') {
    siteEditData.value = { ...data }
    siteEditVisible.value = true
    return
  }

  // Handle batch edit dialogs
  if (type === 'batch-cname-domain' || type === 'batch-cname-mode' || type === 'batch-node-group') {
     const ids = selectedSites.value.map(s => s.id)
     if (ids.length === 0) {
         ElMessage.warning('请先选择站点')
         return
     }
     batchEditMode.value = type.replace('batch-', '')
     batchEditIds.value = ids
     batchEditVisible.value = true
     return
  }

  if (type === 'batch-edit') {
    const ids = selectedSites.value.map(s => s.id)
    if (ids.length === 0) {
      ElMessage.warning('请先选择站点')
      return
    }
    batchSettingsIds.value = ids
    batchSettingsVisible.value = true
    return
  }

  if (type === 'apply-cert') {
    const ids = selectedSites.value.map(s => s.id)
    if (ids.length === 0) {
      ElMessage.warning('请先选择站点')
      return
    }
    applyCertRows.value = selectedSites.value.map(item => ({ ...item }))
    applyCertVisible.value = true
    return
  }

  const ids = data ? [data.id] : selectedSites.value.map(s => s.id)
  if (ids.length === 0) {
      ElMessage.warning('请先选择站点')
      return
  }
  
  if (type.endsWith('delete')) {
    await ElMessageBox.confirm('确定删除吗？', '提示')
    await request.post('/sites/batch_action', { action: 'delete', ids })
  } else if (type.endsWith('enable') || type.endsWith('disable')) {
    const action = type.split('-').pop()
    await request.post('/sites/batch_action', { action, ids })
  } else if (type.endsWith('unlock')) {
     await request.post('/sites/batch_action', { action: 'unlock', ids })
  } else if (type.endsWith('clear_cache')) {
     await ElMessageBox.confirm('确定清空缓存吗？系统将创建任务并分发到所有节点执行。', '提示', { type: 'warning' })
     const res = await request.post('/sites/batch_action', { action: 'clear_cache', ids })
     const taskId = res?.data?.task_id || res?.task_id
     if (taskId) {
       taskMonitorId.value = String(taskId)
       taskMonitorTitle.value = `清空缓存任务 #${taskId}`
       taskMonitorVisible.value = true
     }
     ElMessage.success(taskId ? `已创建清空缓存任务：${taskId}` : '已提交清空缓存任务')
     return
  }
  ElMessage.success('操作成功')
  fetchSites()
}


const handleExport = () => {
  window.open(`${request.defaults.baseURL}/sites/export`, '_blank')
}

const applyAdvancedSearch = () => {
  advancedVisible.value = false
  fetchSites()
}

onMounted(() => {
  fetchSites()
  loadDomainUsage()
})

const loadDomainUsage = async () => {
  if (isAdmin.value) return
  try {
    const pkgRes = await request.get('/user_packages', { params: { pageSize: 1000 } })
    const list = pkgRes.data?.list || pkgRes.list || []
    if (!list.length) return
    const pkgId = list[0].id
    const usageRes = await request.get('/domain_usage', { params: { user_package_id: pkgId } })
    const usage = usageRes.data || usageRes
    createDisabled.value = !!usage.exceeded
    createDisabledTip.value = usage.message || ''
  } catch (e) {
    createDisabled.value = false
    createDisabledTip.value = ''
  }
}

const handleTaskCompleted = (t) => {
  if (!t) return
  const state = String(t.state || '').toLowerCase()
  if (state === 'fail') {
    ElMessage.error('任务执行失败，请查看任务日志')
  } else if (state === 'done') {
    ElMessage.success('任务执行完成')
  }
}

const applyCertWildcardCount = computed(() => {
  return applyCertRows.value.filter(row => {
    const domains = Array.isArray(row.domains) ? row.domains : []
    return domains.some(d => String(d || '').trim().startsWith('*.'))
  }).length
})

const submitApplyCert = async () => {
  if (applyCertSubmitting.value) return
  const ids = applyCertRows.value.map(s => s.id).filter(Boolean)
  if (!ids.length) {
    ElMessage.warning('请先选择站点')
    return
  }
  applyCertSubmitting.value = true
  try {
    const res = await request.post('/sites/apply_cert', { ids })
    const payload = res?.data || res || {}
    const created = Array.isArray(payload.created_ids) ? payload.created_ids : []
    const skipped = Array.isArray(payload.skipped) ? payload.skipped : []
    if (created.length > 0) {
      ElMessage.success(`证书申请已提交：${created.length}个`)
    }
    if (skipped.length > 0) {
      const msg = skipped
        .map(item => `站点${item.site_id || '-'}：${item.reason || '已忽略'}`)
        .join('\n')
      ElMessage.warning(msg)
    }
    applyCertVisible.value = false
    fetchSites()
  } finally {
    applyCertSubmitting.value = false
  }
}

const normalizeCname = (value) => {
  return (value || '').trim().replace(/\.$/, '').toLowerCase()
}

const extractPrimaryDomain = (row) => {
  const display = row?.domain_display || row?.domainDisplay
  if (display && typeof display === 'string') {
    const first = display.split(',').map(s => s.trim()).filter(Boolean)[0]
    if (first) return first
  }
  if (Array.isArray(row?.domains) && row.domains.length > 0) return row.domains[0]
  return ''
}

const handleBatchEditSuccess = async (payload) => {
  await fetchSites()

  if (payload?.mode !== 'cname-domain') return
  const ids = Array.isArray(payload?.ids) ? payload.ids : []
  if (!ids.length) return

  // Resolve check is best-effort and runs only for current list rows.
  const rows = ids
    .map((id) => siteList.value.find((s) => String(s.id) === String(id)))
    .filter(Boolean)

  if (!rows.length) return

  resolveCheckVisible.value = true
  resolveCheckLoading.value = true
  resolveCheckResults.value = []

  const results = []
  const batchSize = 5
  for (let i = 0; i < rows.length; i += batchSize) {
    const chunk = rows.slice(i, i + batchSize)
    // eslint-disable-next-line no-await-in-loop
    await Promise.all(chunk.map(async (row) => {
      const domain = extractPrimaryDomain(row)
      const expectedCname = row.cname || row.cname_hostname || '-'
      if (!domain) {
        results.push({ domain: '-', expectedCname, resolvedCname: '-', ips: '', ok: false, error: '域名为空' })
        return
      }
      try {
        const res = await request.get('/sites/resolve', { params: { domain }, skipLoading: true })
        const resolvedCname = res?.cname || ''
        const ips = Array.isArray(res?.ips) ? res.ips.join(', ') : ''
        const ok = normalizeCname(resolvedCname) === normalizeCname(expectedCname)
        results.push({
          domain,
          expectedCname,
          resolvedCname: resolvedCname || '-',
          ips,
          ok,
          error: ok ? '' : 'CNAME不匹配'
        })
      } catch (e) {
        results.push({ domain, expectedCname, resolvedCname: '-', ips: '', ok: false, error: '查询失败' })
      }
    }))
  }

  resolveCheckResults.value = results
  resolveCheckLoading.value = false

  const failed = results.filter(r => !r.ok)
  if (failed.length === 0) {
    ElMessage.success('解析检测通过')
  } else {
    ElMessage.warning(`解析检测异常：${failed.length} 个域名未生效`)
  }
}
</script>

<style scoped>
.apply-cert-summary {
  color: var(--el-text-color-primary);
  margin-bottom: 10px;
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.apply-cert-alert {
  margin-bottom: 10px;
}

.apply-cert-list {
  max-height: 280px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  padding: 8px 10px;
}

.apply-cert-item {
  display: flex;
  gap: 8px;
  padding: 6px 2px;
  line-height: 1.4;
  color: var(--el-text-color-primary);
}

.apply-cert-name {
  min-width: 58px;
  color: var(--el-text-color-secondary);
}

.site-list-dialog :deep(.el-dialog__body),
.site-list-dialog :deep(.el-dialog__footer) {
  background: var(--el-bg-color);
}
</style>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
.custom-tabs :deep(.el-tabs__item) { font-weight: 600; }
</style>
