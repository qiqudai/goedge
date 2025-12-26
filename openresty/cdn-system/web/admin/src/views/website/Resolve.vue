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
        <el-button type="primary" @click="syncResolve">同步解析</el-button>
      </div>
      <div class="filter-right">
        <el-select v-model="listQuery.searchField" class="filter-item" style="width: 120px;">
          <el-option label="全字段" value="all" />
          <el-option label="域名" value="domain" />
          <el-option label="CNAME" value="cname" />
        </el-select>
        <el-input
          v-model="listQuery.keyword"
          placeholder="输入域名, 模糊搜索"
          style="width: 260px;"
          class="filter-item"
          @keyup.enter="handleFilter"
        />
        <el-button type="primary" class="filter-item" @click="handleFilter">查询</el-button>
      </div>
    </div>

    <el-table
      v-loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      style="width: 100%;"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column prop="id" label="ID" width="90" />
      <el-table-column prop="site_id" label="网站ID" width="100" />
      <el-table-column prop="domain" label="域名" min-width="200" show-overflow-tooltip />
      <el-table-column prop="cname" label="CNAME" min-width="200" show-overflow-tooltip />
      <el-table-column label="解析状态" width="140">
        <template #default="{ row }">
          <span class="status-dot" :class="statusClass(row.resolveStatus)" />
          <span class="status-text">{{ statusText(row.resolveStatus) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="dns_name" label="DNS API" width="140" />
      <el-table-column prop="task_status" label="任务状态" width="120" />
    </el-table>

    <div class="pagination-container">
      <el-pagination
        v-model:current-page="listQuery.page"
        v-model:page-size="listQuery.pageSize"
        :page-sizes="[10, 20, 30, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'
import { useRouter } from 'vue-router'

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
const dnsMap = ref({})
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
    router.push('/global/default')
  } else if (tab.paneName === 'dns') {
    router.push('/node/dns')
  }
}

const statusText = status => {
  if (status === 'checking') return '检测中'
  if (status === 'success') return '正常'
  if (status === 'failed') return '异常'
  return '未检测'
}

const statusClass = status => {
  if (status === 'checking') return 'status-checking'
  if (status === 'success') return 'status-success'
  if (status === 'failed') return 'status-failed'
  return 'status-default'
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
    const rows = res.list || res.data || []
    list.value = rows.map(site => {
      const domain = site.domain_display ? site.domain_display.split(',')[0] : (site.domains && site.domains[0]) || ''
      return {
        id: site.id,
        site_id: site.id,
        domain,
        cname: site.cname || '-',
        resolveStatus: autoResolve ? 'checking' : 'default',
        dns_name: dnsMap.value[site.dns_provider_id] || '未配置',
        task_status: ''
      }
    })
    total.value = res.total || 0
    listLoading.value = false
    if (autoResolve) {
      runResolve(list.value)
    }
  }).catch(() => {
    listLoading.value = false
  })
}

const runResolve = async rows => {
  resolveRunId += 1
  const currentId = resolveRunId
  for (const row of rows) {
    if (currentId !== resolveRunId) {
      return
    }
    if (!row.domain) {
      row.resolveStatus = 'failed'
      continue
    }
    row.resolveStatus = 'checking'
    try {
      const res = await request.get('/sites/resolve', { params: { domain: row.domain } })
      const resolved = normalizeCname(res.cname)
      const expected = normalizeCname(row.cname)
      row.resolveStatus = resolved && expected && resolved === expected ? 'success' : 'failed'
    } catch (err) {
      row.resolveStatus = 'failed'
    }
  }
}

const syncResolve = () => {
  runResolve(list.value)
}

const handleFilter = () => {
  listQuery.page = 1
  fetchList(true)
}

const handleSizeChange = () => {
  listQuery.page = 1
  fetchList(true)
}

const handlePageChange = page => {
  listQuery.page = page
  fetchList(true)
}

const loadDnsProviders = () => {
  return request.get('/dns/providers').then(res => {
    const listData = res.data?.list || res.list || []
    const mapping = {}
    listData.forEach(item => {
      mapping[item.id] = item.name
    })
    dnsMap.value = mapping
  })
}

onMounted(() => {
  loadDnsProviders().finally(() => {
    fetchList(true)
  })
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
.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
}
.status-text {
  vertical-align: middle;
}
.status-checking {
  background: #e6a23c;
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
</style>



