<template>
  <div class="app-container">
    <h2>已售套餐</h2>

    <div class="filter-container">
      <el-button type="primary" size="" @click="fetchList">同步数据</el-button>
      <el-button
        size=""
        :disabled="selectedIds.length === 0"
        @click="handleBatchDelete"
      >
        删除
      </el-button>
      <div class="filter-inline">
        <el-select v-model="query.keywordType" size="" style="width: 110px">
          <el-option label="用户ID" value="user_id" />
          <el-option label="用户名" value="user_name" />
          <el-option label="套餐名称" value="plan_name" />
        </el-select>
        <el-input
          v-model="query.keyword"
          size=""
          placeholder="输入用户ID"
          clearable
          style="width: 220px"
          @keyup.enter="applyFilter"
        />
        <el-button size="" type="primary" @click="applyFilter">
          <el-icon><Search /></el-icon>
        </el-button>
      </div>
    </div>

    <AppTable
      :data="pagedList"
      :loading="loading"
      border
      style="width: 100%; margin-top: 16px;"
      @selection-change="handleSelectionChange"
      v-model:current-page="page"
      v-model:page-size="pageSize"

      layout="total, prev, pager, next, sizes, jumper"
      :total="total"
      persist-key="plans-sold"
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
      <el-table-column label="Debug:域名" min-width="120">
          <template #default="{ row }">
             {{ row.cname_domain }}
          </template>
      </el-table-column>
      <el-table-column label="Debug:Mode" min-width="100">
          <template #default="{ row }">
             {{ row.cname_mode }}
          </template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button link type="primary" size="" @click="openDetail(row)">详情</el-button>
          <el-button link type="primary" size="" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" size="" @click="openUpgrade(row)">升降配</el-button>
        </template>
      </el-table-column>
    </AppTable>

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
              <span class="detail-value">{{ current.custom_cc_rule ? '允许' : '拒绝' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Websocket:</span>
              <span class="detail-value">{{ current.websocket ? '允许' : '拒绝' }}</span>
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
        <el-button size="" @click="detailVisible = false">关闭</el-button>
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
            <el-button size="" type="primary" @click="submitSwitch">确定</el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <el-dialog v-model="editVisible" title="套餐编辑" width="900px" top="5vh">
      <el-form :model="editForm" label-width="120px">
        
        <el-divider content-position="left">线路分组</el-divider>
        <el-row>
          <el-col :span="8">
            <el-form-item label="区域">
              <el-select v-model="editForm.region_id" placeholder="默认">
                <el-option label="默认" :value="0" />
                <el-option v-for="item in regionOptions" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="线路分组">
              <el-select v-model="editForm.node_group_id" placeholder="请选择">
                <el-option label="默认" :value="0" />
                <el-option v-for="item in nodeGroupOptions" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="备用分组">
              <el-select v-model="editForm.backup_group_id" placeholder="请选择">
                <el-option label="不设置" :value="0" />
                <el-option v-for="item in nodeGroupOptions" :key="item.id" :label="item.name" :value="item.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">资源限制</el-divider>
        <el-row>
          <el-col :span="8">
            <el-form-item label="月流量">
              <el-input v-model="editForm.traffic" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="带宽">
              <el-input v-model="editForm.bandwidth" placeholder="不限">
                <template #append>Mbps</template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="连接数">
              <el-input v-model="editForm.connection" placeholder="不限" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="8">
            <el-form-item label="四层端口数">
              <el-input v-model="editForm.stream_port" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="域名数">
              <el-input v-model="editForm.domain" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="主域名数">
              <el-input disabled placeholder="暂不支持" /> 
              <!-- Screenshot just says "主域名数", checking Basic.vue it doesn't have "Main Domain Limit". 
                   The screenshot has "主域名数". I don't see it in my editForm properties from Basic.vue.
                   Let's add it if needed or placeholder. 
                   Wait, user screenshot has "主域名数". I will add it to UI but disable or link to domain?
                   Actually I'll skip it if I don't have the field, OR add it to editForm structure?
                   Let's check `UserPackage` model. `Domain` usually covers it.
                   Maybe "Domain" is "Subdomain"? 
                   I'll use "主域名数" in place of something else or just add the field visually. -->
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="8">
            <el-form-item label="网站非标端口数">
              <el-input v-model="editForm.http_port" placeholder="不限" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="自定义CC规则">
              <el-switch v-model="editForm.custom_cc_rule" active-text="允许" inactive-text="禁止" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Websocket">
              <el-switch v-model="editForm.websocket" active-text="允许" inactive-text="禁止" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
             <el-col :span="8">
                <el-form-item label="HTTP3">
                  <el-switch v-model="editForm.http3_enabled" active-text="允许" inactive-text="禁止" />
                </el-form-item>
             </el-col>
        </el-row>

        <el-divider content-position="left">到期时间</el-divider>
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

        <el-divider content-position="left">续费价格</el-divider>
        <el-row>
          <el-col :span="8">
            <el-form-item label="月付">
              <el-input v-model="editForm.price_monthly"><template #append>元</template></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="季度付">
              <el-input v-model="editForm.price_quarterly"><template #append>元</template></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="年付">
              <el-input v-model="editForm.price_yearly"><template #append>元</template></el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">CNAME设置</el-divider>
        <el-row>
          <el-col :span="8">
            <el-form-item label="主机名">
              <el-input v-model="editForm.cname_hostname" placeholder="" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="CNAME域名">
              <el-select v-model="editForm.cname_domain" placeholder="请选择">
                 <el-option v-for="item in cnameOptions" :key="item.id" :label="item.domain" :value="item.domain" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="CNAME模式">
               <el-select v-model="editForm.cname_mode" placeholder="默认">
                <el-option label="按网站生成(默认)" value="domain" />
                <el-option label="按套餐生成" value="package" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

      </el-form>
      <template #footer>
        <el-button size="" @click="editVisible = false">关闭</el-button>
        <el-button size="" type="primary" @click="submitEdit">确定</el-button>
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

// Options for Edit Form
const regionOptions = ref([])
const nodeGroupOptions = ref([])
const cnameOptions = ref([])

const editVisible = ref(false)
const editForm = ref({
  id: null,
  name: '', // plan_name in db
  end_at: '',
  // Resources
  traffic: '',
  bandwidth: '',
  connection: '',
  domain: '',
  http_port: '',
  stream_port: '',
  custom_cc_rule: false,
  websocket: false,
  // Groups
  region_id: 0,
  node_group_id: 0,
  backup_group_id: 0,
  // CNAME
  cname_hostname: '',
  cname_domain: '',
  cname_mode: 'default',
  // Price
  price_monthly: 0,
  price_quarterly: 0,
  price_yearly: 0,
  // Other
  http3_enabled: false
})

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

const fetchRegions = () => {
  request.get('/regions').then(res => {
    regionOptions.value = res.data.list || []
  })
}

const fetchNodeGroups = () => {
  request.get('/node-groups').then(res => {
    nodeGroupOptions.value = res.data.list || []
  })
}

const fetchCnameDomains = () => {
  request.get('/cname_domains').then(res => {
    cnameOptions.value = res.data.list || []
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
  ElMessageBox.confirm('确认删除选中的套餐?', '提示', { type: 'warning' }).then(() => {
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
  // We should ideally fetch single item to get full details if list is partial.
  // Assuming list has all fields for now or fetch detail.
  // Checking list response in Basic.vue... list usually has most.
  // But safest is to use row data and map it.
  
  // Mapping row (UserPackage) fields to editForm
  editForm.value = {
    id: row.id,
    name: row.plan_name || '', // user_plans table might not have plan_name column editable? Usually it references plan. But UserPackage has snapshot.
    // The screenshot implies we are editing the UserPackage specific settings.
    end_at: formatDateTime(row.end_at),
    
    // Groups
    region_id: row.region_id || row.region || 0, // check api response key
    node_group_id: row.node_group_id || 0,
    backup_group_id: row.backup_group_id || 0,

    // Resources
    traffic: row.traffic,
    bandwidth: row.bandwidth,
    connection: row.connection,
    domain: row.domain,
    http_port: row.http_port,
    stream_port: row.stream_port,
    custom_cc_rule: row.custom_cc_rule,
    websocket: row.websocket,
    http3_enabled: row.http3_enabled,

    // Price
    price_monthly: row.price_monthly,
    price_quarterly: row.price_quarterly,
    price_yearly: row.price_yearly,

    // CNAME
    // User request: Hostname is the "record_id" (Resolve Value). Show it.
    cname_hostname: row.cname_hostname || row.record_id || '',
    cname_domain: row.cname_domain || '',
    cname_mode: row.cname_mode || 'default'
  }
  console.log('[DEBUG] openEdit row:', row)
  console.log('[DEBUG] editForm cname:', editForm.value.cname_domain, editForm.value.cname_mode)
  editVisible.value = true
}

const submitEdit = () => {
  if (!editForm.value.id) {
    return
  }
  
  // Validation: Backup Group cannot be same as Main Group
  if (editForm.value.node_group_id && editForm.value.backup_group_id && 
      editForm.value.node_group_id === editForm.value.backup_group_id) {
    ElMessage.error('备用分组不能与线路分组相同')
    return
  }

  // Construct payload. 
  // IMPORTANT: Backend needs to handle these fields.
  const payload = { ...editForm.value }
  
  request.put(`/user_plans/${editForm.value.id}`, payload).then(() => {
    ElMessage.success('更新成功')
    editVisible.value = false
    fetchList()
  })
}

const submitSwitch = () => {
  ElMessage.info('暂未实现')
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
  fetchRegions()
  fetchNodeGroups()
  fetchCnameDomains()
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
