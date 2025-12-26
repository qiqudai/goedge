<template>
  <div class="app-container">
    <h2>已售套餐</h2>

    <div class="filter-container">
      <el-button type="primary" size="normal" @click="fetchList">同步数据</el-button>
      <el-button
        size="normal"
        :disabled="selectedIds.length === 0"
        @click="handleBatchDelete"
      >
        删除
      </el-button>
      <div class="filter-inline">
        <el-select v-model="query.keywordType" size="normal" style="width: 110px">
          <el-option label="用户ID" value="user_id" />
          <el-option label="用户名" value="user_name" />
          <el-option label="套餐名称" value="plan_name" />
        </el-select>
        <el-input
          v-model="query.keyword"
          size="normal"
          placeholder="输入用户ID"
          clearable
          style="width: 220px"
          @keyup.enter="applyFilter"
        />
        <el-button size="normal" type="primary" @click="applyFilter">
          <el-icon><Search /></el-icon>
        </el-button>
      </div>
    </div>

    <el-table
      :data="pagedList"
      v-loading="loading"
      border
      style="width: 100%; margin-top: 16px;"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="50" />
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="用户" min-width="160">
        <template #default="{ row }">
          {{ formatUser(row) }}
        </template>
      </el-table-column>
      <el-table-column label="基础套餐" min-width="160">
        <template #default="{ row }">
          {{ formatPackage(row) }}
        </template>
      </el-table-column>
      <el-table-column label="套餐名称" min-width="160">
        <template #default="{ row }">
          {{ row.plan_name || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="解析值" min-width="140">
        <template #default="{ row }">
          {{ row.record_id || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="购买时间" min-width="170">
        <template #default="{ row }">
          {{ formatPurchaseTime(row) }}
        </template>
      </el-table-column>
      <el-table-column label="到期时间" min-width="170">
        <template #default="{ row }">
          {{ formatDateTime(row.end_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button link type="primary" size="normal" @click="openDetail(row)">详情</el-button>
          <el-button link type="primary" size="normal" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" size="normal" @click="openUpgrade(row)">升降配</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-container">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, prev, pager, next, sizes, jumper"
        :total="total"
      />
    </div>

    <el-dialog v-model="detailVisible" title="套餐详情" width="720px">
      <el-tabs v-model="detailTab">
        <el-tab-pane label="使用情况" name="usage">
          <el-table :data="usageRows" border>
            <el-table-column prop="label" label="" width="160" />
            <el-table-column prop="total" label="总额度" />
            <el-table-column prop="used" label="已使用" />
            <el-table-column prop="remain" label="剩余" />
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="套餐详情" name="detail">
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-label">名称:</span>
              <span class="detail-value">{{ current.plan_name || '-' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">流量(GB):</span>
              <span class="detail-value">{{ formatLimit(current.traffic) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">带宽:</span>
              <span class="detail-value">{{ formatText(current.bandwidth) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">连接数:</span>
              <span class="detail-value">{{ formatLimit(current.connection) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">域名数:</span>
              <span class="detail-value">{{ formatLimit(current.domain) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">HTTP端口数:</span>
              <span class="detail-value">{{ formatLimit(current.http_port) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">转发端口数:</span>
              <span class="detail-value">{{ formatLimit(current.stream_port) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">自定义CC规则:</span>
              <span class="detail-value">{{ current.custom_cc_rule ? '允许' : '禁止' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Websocket:</span>
              <span class="detail-value">{{ current.websocket ? '允许' : '禁止' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">到期时间:</span>
              <span class="detail-value">{{ formatDateTime(current.end_at) }}</span>
            </div>
          </div>
          <div class="detail-section">
            <div class="detail-title">已购升级包</div>
            <el-table :data="upgradeRows" border>
              <el-table-column prop="name" label="名称" />
              <el-table-column prop="amount" label="升级包" />
              <el-table-column prop="total" label="总数" />
            </el-table>
          </div>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button size="normal" @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="upgradeVisible" title="升降配" width="720px">
      <el-tabs v-model="upgradeTab">
        <el-tab-pane label="升级包" name="upgrade">
          <el-table :data="upgradeRows" border>
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="amount" label="升级包" />
            <el-table-column prop="total" label="总数" />
            <el-table-column label="操作" width="120">
              <template #default>
                <span class="empty-text">暂无数据</span>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="更换套餐" name="switch">
          <el-form label-width="80px">
            <el-form-item label="选择套餐">
              <el-select v-model="upgradeForm.planId" placeholder="请选择" style="width: 100%;">
                <el-option
                  v-for="plan in planOptions"
                  :key="plan.id"
                  :label="plan.name"
                  :value="plan.id"
                />
              </el-select>
            </el-form-item>
          </el-form>
          <div class="dialog-footer">
            <el-button size="normal" type="primary" @click="submitSwitch">确定</el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <el-dialog v-model="editVisible" title="套餐编辑" width="520px">
      <el-form label-width="90px">
        <el-form-item label="套餐名称">
          <el-input v-model="editForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker
            v-model="editForm.end_at"
            type="datetime"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="请选择"
            clearable
            style="width: 100%;"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="normal" @click="editVisible = false">关闭</el-button>
        <el-button size="normal" type="primary" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import request from '@/utils/request'

const list = ref([])
const loading = ref(false)
const selectedIds = ref([])
const query = ref({ keywordType: 'user_id', keyword: '' })
const page = ref(1)
const pageSize = ref(10)

const detailVisible = ref(false)
const detailTab = ref('usage')
const current = ref({})
const upgradeRows = ref([])

const upgradeVisible = ref(false)
const upgradeTab = ref('upgrade')
const upgradeForm = ref({ planId: '' })
const planOptions = ref([])

const editVisible = ref(false)
const editForm = ref({ id: null, name: '', end_at: '' })

const fetchList = () => {
  loading.value = true
  request.get('/user_plans').then((res) => {
    list.value = res.data.list || []
  }).finally(() => {
    loading.value = false
  })
}

const fetchPlans = () => {
  request.get('/plans').then((res) => {
    planOptions.value = res.data.list || []
  })
}

const filteredList = computed(() => {
  const keyword = query.value.keyword.trim()
  if (!keyword) {
    return list.value
  }
  const lower = keyword.toLowerCase()
  return list.value.filter((item) => {
    if (query.value.keywordType === 'user_id') {
      return String(item.user_id || '').includes(keyword)
    }
    if (query.value.keywordType === 'user_name') {
      return String(item.user_name || '').toLowerCase().includes(lower)
    }
    if (query.value.keywordType === 'plan_name') {
      return String(item.plan_name || '').toLowerCase().includes(lower)
    }
    return false
  })
})

const total = computed(() => filteredList.value.length)

const pagedList = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredList.value.slice(start, start + pageSize.value)
})

const applyFilter = () => {
  page.value = 1
}

const handleSelectionChange = (rows) => {
  selectedIds.value = rows.map((row) => row.id)
}

const handleBatchDelete = () => {
  if (selectedIds.value.length === 0) {
    return
  }
  ElMessageBox.confirm('确认删除所选套餐吗?', '提示', { type: 'warning' }).then(() => {
    request({
      url: '/user_plans',
      method: 'delete',
      data: { ids: selectedIds.value }
    }).then(() => {
      ElMessage.success('删除成功')
      selectedIds.value = []
      fetchList()
    })
  })
}

const openDetail = (row) => {
  current.value = { ...row }
  detailTab.value = 'usage'
  detailVisible.value = true
}

const openUpgrade = (row) => {
  current.value = { ...row }
  upgradeTab.value = 'upgrade'
  upgradeForm.value = { planId: '' }
  upgradeVisible.value = true
}

const openEdit = (row) => {
  editForm.value = {
    id: row.id,
    name: row.plan_name || '',
    end_at: formatDateTime(row.end_at)
  }
  editVisible.value = true
}

const submitEdit = () => {
  if (!editForm.value.id) {
    return
  }
  request.put(`/user_plans/${editForm.value.id}`, {
    name: editForm.value.name,
    end_at: editForm.value.end_at
  }).then(() => {
    ElMessage.success('保存成功')
    editVisible.value = false
    fetchList()
  })
}

const submitSwitch = () => {
  ElMessage.info('暂无可用套餐')
}

const usageRows = computed(() => {
  const traffic = buildUsage(formatLimit(current.value.traffic))
  const domain = buildUsage(formatLimit(current.value.domain))
  const httpPort = buildUsage(formatLimit(current.value.http_port))
  const streamPort = buildUsage(formatLimit(current.value.stream_port))
  return [
    { label: '流量(GB)', ...traffic },
    { label: '域名数', ...domain },
    { label: 'HTTP端口数', ...httpPort },
    { label: '转发端口数', ...streamPort }
  ]
})

const buildUsage = (total) => {
  if (total === '不限') {
    return { total: '不限', used: 0, remain: '不限' }
  }
  return { total, used: 0, remain: total }
}

const formatUser = (row) => {
  const name = row.user_name ? String(row.user_name).trim() : ''
  if (!name) {
    return `ID: ${row.user_id || '-'}`
  }
  return `${name} (${row.user_id})`
}

const formatPackage = (row) => {
  if (!row.package_id) {
    return '-'
  }
  const name = row.package_name || '-'
  return `${name} (id: ${row.package_id})`
}

const formatPurchaseTime = (row) => {
  return formatDateTime(row.start_at || row.created_at)
}

const formatText = (val) => {
  if (val === null || val === undefined || val === '') {
    return '不限'
  }
  return val
}

const formatLimit = (val) => {
  if (val === null || val === undefined || val === '') {
    return '不限'
  }
  if (typeof val === 'number' && val <= 0) {
    return '不限'
  }
  return val
}

const formatDateTime = (value) => {
  if (!value) {
    return '-'
  }
  let date = new Date(value)
  if (Number.isNaN(date.getTime()) && typeof value === 'string') {
    date = new Date(value.replace(' ', 'T'))
  }
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }
  const pad = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

onMounted(() => {
  fetchList()
  fetchPlans()
})
</script>

<style scoped>
.filter-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-inline {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pagination-container {
  margin-top: 16px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 24px;
  row-gap: 12px;
  margin-bottom: 16px;
}

.detail-item {
  display: flex;
  gap: 8px;
}

.detail-label {
  color: #606266;
  min-width: 110px;
}

.detail-section {
  margin-top: 16px;
}

.detail-title {
  margin: 8px 0 12px;
  color: #909399;
}

.dialog-footer {
  display: flex;
  justify-content: flex-start;
  padding-top: 8px;
}

.empty-text {
  color: #909399;
}
</style>
