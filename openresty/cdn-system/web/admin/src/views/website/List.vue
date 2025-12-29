<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="activeTab" class="custom-tabs">
        <el-tab-pane label="网站列表" name="list">
          <SiteTable
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
          <DefaultSettings :is-admin="isAdmin" />
        </el-tab-pane>

        <el-tab-pane label="DNS API" name="dns">
          <DnsApiList />
        </el-tab-pane>

        <el-tab-pane label="解析检测" name="resolve">
          <ResolvePage :hide-tabs="true" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- Global Advanced Search -->
    <el-dialog v-model="advancedVisible" title="高级搜索" width="500px">
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
      @success="fetchSites"
    />
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

import SiteTable from './list/SiteTable.vue'
import SiteEditDialog from './list/SiteEditDialog.vue'
import BatchEditDialog from './list/BatchEditDialog.vue'
import DefaultSettings from './list/DefaultSettings.vue'
import DnsApiList from './list/DnsApiList.vue'
import ResolvePage from './Resolve.vue'

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

const batchEditVisible = ref(false)
const batchEditMode = ref('')
const batchEditIds = ref([])

const advancedVisible = ref(false)
const advQuery = reactive({ status: '' })

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
     // TODO: Implement clear cache
     ElMessage.info('暂未实现')
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
})
</script>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
.custom-tabs :deep(.el-tabs__item) { font-weight: 600; }
</style>
