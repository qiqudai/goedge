<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane label="日志查询" name="query" />
      <el-tab-pane label="申请记录" name="history" />
    </el-tabs>

    <div v-if="activeTab === 'query'">
      <div class="filter-container">
        <div class="filter-row">
          <el-select
            v-model="listQuery.domain"
            placeholder="所有域名"
            clearable
            filterable
            style="width: 200px"
            class="filter-item"
          >
            <el-option v-for="item in domainOptions" :key="item" :label="item" :value="item" />
          </el-select>
          <el-input
            v-model="listQuery.keyword"
            placeholder="域名/URI/IP"
            clearable
            style="width: 220px"
            class="filter-item"
          />
          <el-button type="primary" class="filter-item" :icon="Search" @click="handleFilter">搜索</el-button>
          <el-button class="filter-item" @click="handleDownload">申请下载</el-button>
          <el-button link class="filter-item" @click="toggleAdvanced">
            高级搜索
            <el-icon class="el-icon--right">
              <ArrowDown v-if="!advancedVisible" />
              <ArrowUp v-else />
            </el-icon>
          </el-button>
        </div>

        <div v-show="advancedVisible" class="advanced-filter-area">
          <el-form :inline="true" label-position="right" label-width="80px">
            <el-form-item label="时间范围">
              <el-date-picker
                v-model="listQuery.timeRange"
                type="datetimerange"
                range-separator="至"
                start-placeholder="开始时间"
                end-placeholder="结束时间"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                clearable
                style="width: 360px"
              />
            </el-form-item>
            <el-form-item label="域名匹配">
              <el-select v-model="listQuery.domain_mode" clearable placeholder="精确">
                <el-option label="精确" value="exact" />
                <el-option label="模糊" value="fuzzy" />
              </el-select>
            </el-form-item>
            <el-form-item label="客户端IP">
              <el-input v-model="listQuery.client_ip" placeholder="请输入IP地址" clearable />
            </el-form-item>
            <el-form-item label="请求地址">
              <el-input v-model="listQuery.uri" placeholder="不包含域名部分的URI" clearable style="width: 300px">
                <template #prepend>
                  <el-select v-model="listQuery.uri_mode" style="width: 80px">
                    <el-option label="模糊" value="fuzzy" />
                    <el-option label="精确" value="exact" />
                  </el-select>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item label="请求方法">
              <el-select v-model="listQuery.method" clearable placeholder="如GET">
                <el-option label="GET" value="GET" />
                <el-option label="POST" value="POST" />
                <el-option label="HEAD" value="HEAD" />
                <el-option label="PUT" value="PUT" />
                <el-option label="DELETE" value="DELETE" />
                <el-option label="OPTIONS" value="OPTIONS" />
              </el-select>
            </el-form-item>
            <el-form-item label="状态码">
              <el-input v-model="listQuery.status" placeholder="如200" clearable />
            </el-form-item>
            <el-form-item label="状态范围">
              <div class="range-input">
                <el-input v-model="listQuery.status_min" placeholder="最小" clearable />
                <span class="range-sep">-</span>
                <el-input v-model="listQuery.status_max" placeholder="最大" clearable />
              </div>
            </el-form-item>
            <el-form-item label="访问端口">
              <el-input v-model="listQuery.port" placeholder="端口" clearable />
            </el-form-item>
            <el-form-item label="节点ID">
              <el-input v-model="listQuery.node_id" placeholder="节点ID" clearable />
            </el-form-item>
            <el-form-item label="节点IP">
              <el-input v-model="listQuery.node_ip" placeholder="节点IP" clearable />
            </el-form-item>
            <el-form-item label="协议">
              <el-select v-model="listQuery.scheme" clearable placeholder="http/https">
                <el-option label="http" value="http" />
                <el-option label="https" value="https" />
              </el-select>
            </el-form-item>
            <el-form-item label="缓存状态">
              <el-select v-model="listQuery.cache_status" clearable placeholder="状态">
                <el-option label="HIT" value="HIT" />
                <el-option label="MISS" value="MISS" />
                <el-option label="EXPIRED" value="EXPIRED" />
                <el-option label="STALE" value="STALE" />
                <el-option label="BYPASS" value="BYPASS" />
                <el-option label="REVALIDATED" value="REVALIDATED" />
                <el-option label="UPDATING" value="UPDATING" />
              </el-select>
            </el-form-item>
            <el-form-item label="来源">
              <el-input v-model="listQuery.referer" placeholder="Referer" clearable />
            </el-form-item>
            <el-form-item label="浏览器">
              <el-input v-model="listQuery.user_agent" placeholder="User-Agent" clearable />
            </el-form-item>
            <el-form-item label="回源地址">
              <el-input v-model="listQuery.upstream_addr" placeholder="回源地址" clearable />
            </el-form-item>
            <el-form-item label="SSL协议">
              <el-input v-model="listQuery.ssl_protocol" placeholder="SSL协议" clearable />
            </el-form-item>
            <el-form-item label="SSL套件">
              <el-input v-model="listQuery.ssl_cipher" placeholder="SSL套件" clearable />
            </el-form-item>
            <el-form-item>
              <el-button @click="resetFilters">重置</el-button>
              <el-button link @click="advancedVisible = false">收起搜索</el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <div v-if="hasActiveFilters" class="filter-tags-container">
        <el-tag v-if="listQuery.timeRange?.length" closable @close="listQuery.timeRange = []">
          时间范围: {{ listQuery.timeRange[0] }} - {{ listQuery.timeRange[1] }}
        </el-tag>
        <el-tag v-if="listQuery.domain" closable @close="listQuery.domain = ''">域名: {{ listQuery.domain }}</el-tag>
        <el-tag v-if="listQuery.keyword" closable @close="listQuery.keyword = ''">关键字: {{ listQuery.keyword }}</el-tag>
        <el-button link type="primary" size="small" @click="resetFilters">清除</el-button>
      </div>

      <el-table
        v-loading="listLoading"
        :data="list"
        border
        fit
        highlight-current-row
        style="width: 100%; margin-top: 10px;"
        size="small"
      >
        <el-table-column prop="timestamp" label="时间" width="170" show-overflow-tooltip />
        <el-table-column prop="host" label="域名" width="200" show-overflow-tooltip />
        <el-table-column prop="scheme" label="协议" width="70" />
        <el-table-column prop="method" label="方法" width="70" />
        <el-table-column prop="uri" label="URI" min-width="220" show-overflow-tooltip />
        <el-table-column prop="status" label="状态码" width="70">
          <template #default="{ row }">
            <span :class="getStatusColor(row.status)">{{ row.status }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="remote_addr" label="客户端IP" width="130" />
        <el-table-column prop="bytes" label="字节数" width="90" />
        <el-table-column prop="request_time" label="耗时" width="80" />
        <el-table-column prop="upstream_response_time" label="回源耗时" width="90" />
        <el-table-column prop="upstream_addr" label="回源地址" width="140" show-overflow-tooltip />
        <el-table-column prop="upstream_cache_status" label="缓存状态" width="90" />
        <el-table-column prop="http_referer" label="来源" width="140" show-overflow-tooltip />
        <el-table-column prop="http_user_agent" label="浏览器" width="160" show-overflow-tooltip />
        <el-table-column prop="node_id" label="节点ID" width="80" />
        <el-table-column prop="node_ip" label="节点IP" width="130" />
        <el-table-column prop="ssl_protocol" label="SSL协议" width="90" />
        <el-table-column prop="ssl_cipher" label="SSL套件" width="140" show-overflow-tooltip />
      </el-table>

      <div class="pagination-container">
        <el-pagination
          v-model:current-page="listQuery.page"
          v-model:page-size="listQuery.pageSize"

          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="handleFilter"
          @current-change="handleFilter"
        />
      </div>
    </div>

    <div v-else-if="activeTab === 'history'">
      <div class="empty-block">暂无申请记录</div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import request from '@/utils/request'
import { ArrowDown, ArrowUp, Search } from '@element-plus/icons-vue'

const activeTab = ref('query')
const advancedVisible = ref(true)
const listLoading = ref(false)
const list = ref([])
const total = ref(0)
const domainOptions = ref([])

const listQuery = reactive({
  page: 1,
  pageSize: 20,
  domain: '',
  domain_mode: 'exact',
  keyword: '',
  timeRange: [],
  client_ip: '',
  uri: '',
  uri_mode: 'fuzzy',
  method: '',
  status: '',
  status_min: '',
  status_max: '',
  port: '',
  node_id: '',
  node_ip: '',
  scheme: '',
  cache_status: '',
  referer: '',
  user_agent: '',
  upstream_addr: '',
  ssl_protocol: '',
  ssl_cipher: ''
})

const hasActiveFilters = computed(() => {
  return Boolean(
    listQuery.domain ||
      listQuery.keyword ||
      listQuery.timeRange.length ||
      listQuery.client_ip ||
      listQuery.uri ||
      listQuery.status ||
      listQuery.status_min ||
      listQuery.status_max
  )
})

const getStatusColor = status => {
  const code = parseInt(status, 10)
  if (code >= 200 && code < 300) return 'text-success'
  if (code >= 300 && code < 400) return 'text-warning'
  if (code >= 400 && code < 500) return 'text-danger'
  if (code >= 500) return 'text-critical'
  return ''
}

const toggleAdvanced = () => {
  advancedVisible.value = !advancedVisible.value
}

const resetFilters = () => {
  listQuery.domain = ''
  listQuery.domain_mode = 'exact'
  listQuery.keyword = ''
  listQuery.timeRange = []
  listQuery.client_ip = ''
  listQuery.uri = ''
  listQuery.uri_mode = 'fuzzy'
  listQuery.method = ''
  listQuery.status = ''
  listQuery.status_min = ''
  listQuery.status_max = ''
  listQuery.port = ''
  listQuery.node_id = ''
  listQuery.node_ip = ''
  listQuery.scheme = ''
  listQuery.cache_status = ''
  listQuery.referer = ''
  listQuery.user_agent = ''
  listQuery.upstream_addr = ''
  listQuery.ssl_protocol = ''
  listQuery.ssl_cipher = ''
  handleFilter()
}

const handleFilter = () => {
  listLoading.value = true
  request
    .get('/logs/access', { params: listQuery })
    .then(res => {
      list.value = res.data?.list || []
      total.value = res.data?.total || 0
      listLoading.value = false
    })
    .catch(() => {
      listLoading.value = false
    })
}

const handleDownload = () => {
  // TODO: Implement download logic
}

onMounted(() => {
  handleFilter()
})
</script>

<style scoped>
.filter-container {
  margin-bottom: 20px;
}
.filter-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.advanced-filter-area {
  background: #f8f9fa;
  padding: 15px;
  border-radius: 4px;
  margin-bottom: 10px;
  border: 1px solid #ebeef5;
}
.filter-tags-container {
  margin-bottom: 10px;
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
}
.range-input {
  display: flex;
  align-items: center;
  gap: 6px;
}
.range-sep {
  color: #909399;
}
.text-success { color: #67c23a; }
.text-warning { color: #e6a23c; }
.text-danger { color: #f56c6c; }
.text-critical { color: #ff0000; font-weight: bold; }
.pagination-container {
  margin-top: 15px;
  text-align: right;
}
.empty-block {
  text-align: center;
  padding: 50px;
  color: #909399;
}
</style>
