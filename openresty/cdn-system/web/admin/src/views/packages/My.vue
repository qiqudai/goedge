<template>
  <div class="app-container">
    <div class="page-header">
      <div class="page-title">我的套餐</div>
    </div>

    <el-table :data="list" v-loading="loading" border style="width: 100%;">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="套餐名称" min-width="160" show-overflow-tooltip />
      <el-table-column label="到期时间" min-width="160">
        <template #default="{ row }">
          {{ formatDate(row.end_at) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'expired' ? 'danger' : 'success'">
            {{ row.status === 'expired' ? '已到期' : '正常' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" min-width="260">
        <template #default="{ row }">
          <el-button link type="primary" size="" @click="openDetail(row)">详情</el-button>
          <el-button link type="primary" size="" @click="openRenew(row)">续费</el-button>
          <el-button link type="primary" size="" @click="openUpgrade(row)">升降配</el-button>
          <el-button link type="primary" size="" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

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
              <span class="detail-value">{{ current.name || '-' }}</span>
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
              <span class="detail-label">IPv6:</span>
              <span class="detail-value">{{ current.ipv6 ? '开启' : '关闭' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">到期时间:</span>
              <span class="detail-value">{{ formatDate(current.end_at) }}</span>
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

    <el-dialog v-model="renewVisible" title="套餐续费" width="520px">
      <el-form label-width="80px">
        <el-form-item label="时长">
          <el-select v-model="renewForm.period" placeholder="请选择" style="width: 100%;">
            <el-option label="一个月" value="month" />
            <el-option label="三个月" value="quarter" />
            <el-option label="一年" value="year" />
          </el-select>
        </el-form-item>
        <el-form-item label="价格">
          <el-input :model-value="renewPrice" disabled />
        </el-form-item>
        <el-form-item label="优惠码">
          <el-input v-model="renewForm.coupon" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="" @click="renewVisible = false">关闭</el-button>
        <el-button type="primary" size="" @click="submitRenew">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="upgradeVisible" title="升降配" width="520px">
      <el-tabs v-model="upgradeTab">
        <el-tab-pane label="升级包" name="upgrade">
          <div class="empty-block">暂无数据</div>
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
            <el-form-item label="价格">
              <el-input :model-value="switchPrice" disabled />
            </el-form-item>
          </el-form>
          <div class="dialog-footer">
            <el-button size="" @click="upgradeVisible = false">关闭</el-button>
            <el-button type="primary" size="" @click="submitSwitch">确定</el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <el-dialog v-model="editVisible" title="套餐编辑" width="520px">
      <el-form label-width="90px">
        <el-form-item label="套餐名称">
          <el-input v-model="editForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="IPv6开启">
          <el-switch v-model="editForm.ipv6" />
          <span class="form-hint">只针对该套餐生成cname的网站有效</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="" @click="editVisible = false">关闭</el-button>
        <el-button type="primary" size="" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const loading = ref(false)
const planOptions = ref([])

const current = ref({})
const detailVisible = ref(false)
const detailTab = ref('usage')

const renewVisible = ref(false)
const renewForm = ref({ period: '', coupon: '' })

const upgradeVisible = ref(false)
const upgradeTab = ref('switch')
const upgradeForm = ref({ planId: '' })

const editVisible = ref(false)
const editForm = ref({ name: '', ipv6: false })

const upgradeRows = ref([])

const fetchList = () => {
  loading.value = true
  request.get('/user_packages').then((res) => {
    list.value = res.data?.list || res.list || []
  }).finally(() => {
    loading.value = false
  })
}

const loadPlans = () => {
  if (planOptions.value.length) return
  request.get('/plans').then((res) => {
    planOptions.value = res.data?.list || res.list || []
  })
}

const openDetail = (row) => {
  current.value = { ...row }
  detailTab.value = 'usage'
  detailVisible.value = true
}

const openRenew = (row) => {
  current.value = { ...row }
  renewForm.value = { period: '', coupon: '' }
  renewVisible.value = true
}

const openUpgrade = (row) => {
  current.value = { ...row }
  upgradeTab.value = 'switch'
  upgradeForm.value = { planId: '' }
  loadPlans()
  upgradeVisible.value = true
}

const openEdit = (row) => {
  current.value = { ...row }
  editForm.value = { name: row.name || '', ipv6: !!row.ipv6 }
  editVisible.value = true
}

const submitRenew = () => {
  if (!renewForm.value.period) {
    ElMessage.warning('请选择续费时长')
    return
  }
  request.post(`/user_packages/${current.value.id}/renew`, {
    period: renewForm.value.period
  }).then(() => {
    ElMessage.success('续费成功')
    renewVisible.value = false
    fetchList()
  })
}

const submitSwitch = () => {
  if (!upgradeForm.value.planId) {
    ElMessage.warning('请选择套餐')
    return
  }
  request.post(`/user_packages/${current.value.id}/switch`, {
    package_id: upgradeForm.value.planId
  }).then(() => {
    ElMessage.success('操作成功')
    upgradeVisible.value = false
    fetchList()
  })
}

const submitEdit = () => {
  request.put(`/user_packages/${current.value.id}`, {
    name: editForm.value.name,
    ipv6: editForm.value.ipv6
  }).then(() => {
    ElMessage.success('保存成功')
    editVisible.value = false
    fetchList()
  })
}

const formatDate = (val) => {
  if (!val) return '-'
  const date = new Date(val)
  if (Number.isNaN(date.getTime())) return '-'
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  const ss = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${m}-${d} ${hh}:${mm}:${ss}`
}

const formatText = (val) => {
  return val ? String(val) : '不限'
}

const formatLimit = (val) => {
  if (val === null || val === undefined) return '不限'
  if (typeof val === 'number' && val <= 0) return '不限'
  if (String(val).trim() === '') return '不限'
  return val
}

const usageRows = computed(() => {
  const data = current.value || {}
  const rows = [
    { label: '流量(GB)', total: formatLimit(data.traffic), used: 0 },
    { label: '域名数', total: formatLimit(data.domain), used: 0 },
    { label: '主域名数', total: formatLimit(data.domain), used: 0 },
    { label: 'HTTP端口数', total: formatLimit(data.http_port), used: 0 },
    { label: '转发端口数', total: formatLimit(data.stream_port), used: 0 }
  ]
  return rows.map((item) => {
    const total = item.total
    if (total === '不限') {
      return { ...item, remain: '不限' }
    }
    const numericTotal = Number(total)
    const used = Number(item.used || 0)
    const remain = Number.isNaN(numericTotal) ? '-' : Math.max(numericTotal - used, 0)
    return { ...item, remain }
  })
})

const renewPrice = computed(() => {
  const row = current.value || {}
  switch (renewForm.value.period) {
    case 'month':
      return formatPrice(row.month_price)
    case 'quarter':
      return formatPrice(row.quarter_price)
    case 'year':
      return formatPrice(row.year_price)
    default:
      return '-'
  }
})

const switchPrice = computed(() => {
  const plan = planOptions.value.find((item) => item.id === upgradeForm.value.planId)
  if (!plan) return '-'
  const price = plan.price_monthly ?? plan.price_quarterly ?? plan.price_yearly
  return formatPrice(price)
})

const formatPrice = (val) => {
  if (val === null || val === undefined || val === '') return '-'
  return `${val} 元`
}

onMounted(() => {
  fetchList()
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 24px;
  margin-bottom: 16px;
}

.detail-item {
  display: flex;
  align-items: center;
}

.detail-label {
  color: #909399;
  min-width: 90px;
}

.detail-value {
  color: #303133;
}

.detail-section {
  margin-top: 16px;
}

.detail-title {
  font-weight: 600;
  margin-bottom: 8px;
}

.form-hint {
  margin-left: 12px;
  color: #909399;
  font-size: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 12px;
}

.empty-block {
  padding: 32px 0;
  text-align: center;
  color: #909399;
}
</style>
