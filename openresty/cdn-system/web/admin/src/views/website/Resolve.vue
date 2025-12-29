<template>
  <div class="app-container">
    <el-tabs v-if="!hideTabs" v-model="activeTopTab" class="site-tabs" @tab-click="handleTopTab">
      <el-tab-pane label="网站列表" name="list" />
      <el-tab-pane label="默认设置" name="default" />
      <el-tab-pane label="DNS API" name="dns" />
      <el-tab-pane label="解析检测" name="resolve" />
    </el-tabs>

    <div class="filter-container">
  <div class="filter-left">
    <el-button type="primary" :loading="resolving" @click="syncResolve">{{ resolving ? '正在检测中...' : '开始检测' }}</el-button>
  </div>
  <div class="filter-right">
    <el-input
      v-model="listQuery.keyword"
      placeholder="输入域名搜索"
      style="width: 260px;"
      class="filter-item"
      @keyup.enter="handleFilter"
    >
      <template #suffix><el-icon><Search /></el-icon></template>
    </el-input>
    <el-button type="primary" class="filter-item" @click="handleFilter">查询</el-button>
  </div>
</div>

<AppTable
  :data="list"
  :loading="listLoading"
  border
  fit
  highlight-current-row
  style="width: 100%;"
  v-model:current-page="listQuery.page"
  v-model:page-size="listQuery.pageSize"
  layout="total, sizes, prev, pager, next, jumper"
  :total="total"
  @size-change="handleSizeChange"
  @current-change="handlePageChange"
>
  <el-table-column type="selection" width="55" align="center" />
  <el-table-column prop="id" label="ID" width="80" align="center" />
  <el-table-column prop="site_id" label="网站ID" width="100" align="center" />
      <el-table-column label="域名" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span>{{ row.domain }}</span>
          <el-icon class="copy-icon" @click.stop="copyText(row.domain)"><CopyDocument /></el-icon>
        </template>
      </el-table-column>
      <el-table-column label="CNAME" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span>{{ row.cname }}</span>
          <el-icon class="copy-icon" @click.stop="copyText(row.cname)"><CopyDocument /></el-icon>
        </template>
      </el-table-column>
  <el-table-column label="解析状态" width="120" align="center">
    <template #default="{ row }">
      <el-tag v-if="row.resolveStatus === 'checking'" type="warning" size="small">检测中</el-tag>
      <el-tag v-else-if="row.resolveStatus === 'success'" type="success" size="small">正常</el-tag>
      <el-tag v-else-if="row.resolveStatus === 'failed'" type="danger" size="small">异常</el-tag>
      <el-popover v-else-if="row.resolveResult" trigger="hover" placement="top" :width="200">
         <template #default>
            <div style="font-size:12px">
               <div v-if="row.resolveResult.cname">CNAME: {{ row.resolveResult.cname }}</div>
               <div v-if="row.resolveResult.ips">IP: {{ row.resolveResult.ips.join(', ') }}</div>
               <div v-if="row.resolveResult.error" style="color:red">{{ row.resolveResult.error }}</div>
            </div>
         </template>
         <template #reference>
            <el-tag :type="row.resolveStatus === 'success' ? 'success' : 'danger'" size="small">
               {{ row.resolveStatus === 'success' ? '正常' : (row.resolveStatus === 'failed' ? '异常' : '未检测') }}
            </el-tag>
         </template>
      </el-popover>
      <el-tag v-else type="info" size="small">未检测</el-tag>
    </template>
  </el-table-column>
  <el-table-column prop="dns_api" label="DNS API" min-width="150" />
  <el-table-column prop="task_status" label="任务状态" width="100" />
</AppTable>
</div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { Search, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const props = defineProps({
hideTabs: {
type: Boolean,
default: false
}
})

const router = useRouter()
const activeTopTab = ref('resolve')
const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const resolving = ref(false)
let resolveRunId = 0

const listQuery = reactive({
page: 1,
pageSize: 10,
keyword: '',
searchField: 'domain'
})

const handleTopTab = tab => {
if (tab.paneName === 'list') {
router.push('/website/list')
} else if (tab.paneName === 'default') {
router.push('/website/list?tab=default') // Assuming routing handles this, or implement better navigation
} else if (tab.paneName === 'dns') {
router.push('/website/list?tab=dns')
}
}

const copyText = async (text) => {
  if (!text || text === '-') return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success({ message: '复制成功', duration: 1500 })
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const normalizeCname = value => {
return (value || '').trim().replace(/\.$/, '').toLowerCase()
}

const fetchList = (autoResolve = false) => {
listLoading.value = true
request.get('/sites', {
params: {
  page: listQuery.page,
  pageSize: listQuery.pageSize,
  keyword: listQuery.keyword,
  search_field: listQuery.searchField
}
}).then(res => {
const rows = res.data?.list || res.list || []
list.value = rows.map(site => {
  const domain = site.domain_display ? site.domain_display.split(',')[0] : (site.domains && site.domains[0]) || ''
  return {
    id: site.id,
    site_id: site.id,
    domain,
    cname: site.cname || site.cname_hostname || '-',
    resolveStatus: 'default',
    resolveResult: null,
    dns_api: '-', // Placeholder, need backend data
    task_status: '-' // Placeholder
  }
})
total.value = res.data?.total || res.total || 0
listLoading.value = false
if (autoResolve) {
  nextTick(() => {
     runResolve(list.value)
  })
}
}).catch(() => {
listLoading.value = false
})
}

const runResolve = async rows => {
if (resolving.value) return
resolving.value = true
resolveRunId += 1
const currentId = resolveRunId

// Create a queue or run in parallel batches
const batchSize = 5
for (let i = 0; i < rows.length; i += batchSize) {
 if (currentId !== resolveRunId) break
 const chunk = rows.slice(i, i + batchSize)
 await Promise.all(chunk.map(async row => {
    if (!row.domain || row.domain === '-') {
        row.resolveStatus = 'default'
        return
    }
    row.resolveStatus = 'checking'
    try {
        const res = await request.get('/sites/resolve', { params: { domain: row.domain } })
        row.resolveResult = res // { domain, cname, ips }
        
        const resolvedCname = normalizeCname(res.cname)
        const expectedCname = normalizeCname(row.cname)
        
        // Check if CNAME matches OR if resolved IPs match expected (if we knew them)
        // For now, strict CNAME match if expected CNAME is present
        if (expectedCname && expectedCname !== '-') {
            if (resolvedCname === expectedCname) {
                row.resolveStatus = 'success'
            } else {
                // Maybe the resolved CNAME is a sub-cname? 
                // Or maybe it resolved to IP directly?
                // If we have IPs but no CNAME match, it's 'failed' strictly for CNAME setup, 
                // but functionally it might be working if A record is set. 
                // We mark failed for "Resolution Detection" if it doesn't match CNAME config.
                row.resolveStatus = 'failed'
            }
        } else {
            // No expected CNAME, but valid DNS response?
            if (res.ips && res.ips.length > 0) row.resolveStatus = 'success'
            else row.resolveStatus = 'failed'
        }
    } catch (e) {
        row.resolveStatus = 'failed'
        row.resolveResult = { error: '查询失败' }
    }
 }))
}
resolving.value = false
}

const syncResolve = () => {
runResolve(list.value)
}

const handleFilter = () => {
listQuery.page = 1
fetchList()
}

const handleSizeChange = () => {
listQuery.page = 1
fetchList()
}

const handlePageChange = page => {
listQuery.page = page
fetchList()
}

onMounted(() => {
fetchList(true)
})
</script>

<style scoped>
.filter-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.resolve-tip {
    font-size: 13px;
    color: #909399;
    margin-left: 10px;
}
.text-gray { color: #909399; }
.highlight-val { color: var(--el-color-primary); font-weight: 500; }
.copy-icon { margin-left: 5px; cursor: pointer; color: #909399; vertical-align: middle; }
.copy-icon:hover { color: #409eff; }
</style>
