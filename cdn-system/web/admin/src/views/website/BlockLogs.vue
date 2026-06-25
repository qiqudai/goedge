<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <el-tab-pane label="当前封禁" name="current">
        <div class="filter-container">
          <span class="timezone-note">时间显示：本地时区 ({{ localTimeZoneLabel }})</span>
          <el-button type="primary" class="filter-item" @click="handleUnblockBatch">批量解封</el-button>
          <el-button type="danger" class="filter-item" @click="openBlockBatchDialog">加入黑名单</el-button>
          <el-button class="filter-item" @click="handleUnblockSite">解封网站</el-button>
          <el-button class="filter-item" @click="handleExportCurrent">导出当前</el-button>

          <div class="filter-item filter-inline">
            <el-select v-model="currentFilter.type" style="width: 120px" placeholder="类型">
              <el-option label="IP" value="ip" />
              <el-option label="网站ID" value="site_id" />
            </el-select>
            <el-input
              v-model="currentFilter.keyword"
              placeholder="请输入关键词"
              style="width: 200px;"
              class="filter-item"
              @keyup.enter="fetchCurrentList"
            />
            <el-button class="filter-item" type="primary" :icon="Search" @click="fetchCurrentList" />
          </div>
        </div>

        <AppTable
          :data="currentList"
          :loading="loading"
          border
          style="width: 100%"
          :row-key="getCurrentRowKey"
          v-model:current-page="currentQuery.page"
          v-model:page-size="currentQuery.pageSize"

          layout="total, prev, pager, next, sizes, jumper"
          :total="currentTotal"
          persist-key="current"
          @selection-change="handleCurrentSelectionChange"
          @size-change="fetchCurrentList"
          @current-change="fetchCurrentList"
        >
          <el-table-column type="selection" width="55" />
          <el-table-column prop="site_id" label="网站ID" width="100" />
          <el-table-column prop="domain" label="域名" />
          <el-table-column prop="ip" label="IP" />
          <el-table-column prop="location" label="地区" />
          <el-table-column prop="filter" label="规则" />
          <el-table-column prop="block_time" label="封禁时间" />
          <el-table-column prop="release_time" label="解封时间" />
          <el-table-column label="操作" width="100">
            <template #default="scope">
              <el-button link type="primary" size="small" @click="handleUnblock(scope.row)">解封</el-button>
            </template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>

      <el-tab-pane label="统计" name="stats">
        <div class="filter-container">
          <el-radio-group v-model="statsType" style="margin-bottom: 20px;">
            <el-radio-button value="rank">排行</el-radio-button>
          </el-radio-group>
        </div>

        <AppTable
          :data="statsList"
          :loading="loading"
          border
          style="width: 100%"
          v-model:current-page="statsQuery.page"
          v-model:page-size="statsQuery.pageSize"

          layout="total, prev, pager, next, sizes, jumper"
          :total="statsTotal"
          persist-key="stats"
          @size-change="fetchStatsList"
          @current-change="fetchStatsList"
        >
          <el-table-column prop="site_id" label="网站ID" />
          <el-table-column prop="count" label="封禁次数" />
        </AppTable>
      </el-tab-pane>

      <el-tab-pane label="历史记录" name="history">
        <div class="filter-container">
          <el-button class="filter-item" @click="handleExportHistory">导出当前</el-button>

          <div class="filter-item filter-inline">
            <el-dropdown trigger="click" @command="handleHistoryFilterCommand" style="margin-right: 10px;">
              <span class="el-dropdown-link">
                {{ historyFilterLabel }}
                <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="ip">IP</el-dropdown-item>
                  <el-dropdown-item command="site_id">网站ID</el-dropdown-item>
                  <el-dropdown-item command="time_range">时间范围</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-input
              v-if="historyFilter.type !== 'time_range'"
              v-model="historyFilter.keyword"
              placeholder="请输入关键词"
              style="width: 200px;"
              class="filter-item"
              @keyup.enter="fetchHistoryList"
            />
            <el-date-picker
              v-else
              v-model="historyFilter.dateRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
              clearable
              style="width: 360px"
              class="filter-item"
            />
            <el-button class="filter-item" type="primary" :icon="Search" @click="fetchHistoryList" />
          </div>
        </div>

        <AppTable
          :data="historyList"
          :loading="loading"
          border
          style="width: 100%"
          v-model:current-page="historyQuery.page"
          v-model:page-size="historyQuery.pageSize"

          layout="total, prev, pager, next, sizes, jumper"
          :total="historyTotal"
          persist-key="history"
          @size-change="fetchHistoryList"
          @current-change="fetchHistoryList"
        >
          <el-table-column prop="site_id" label="网站ID" width="100" />
          <el-table-column prop="domain" label="域名" />
          <el-table-column prop="ip" label="IP" />
          <el-table-column prop="location" label="地区" />
          <el-table-column prop="filter" label="规则" />
          <el-table-column prop="block_time" label="封禁时间" />
          <el-table-column prop="is_manual" label="解封方式">
            <template #default="scope">{{ scope.row.is_manual ? '\u624b\u52a8' : '\u81ea\u52a8' }}</template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="blockBatchVisible" title="批量加入黑名单" width="560px">
      <el-form label-width="90px">
        <el-form-item label="域名" required>
          <el-input v-model="blockBatchForm.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="IP 列表" required>
          <el-input
            v-model="blockBatchForm.text"
            type="textarea"
            :rows="8"
            placeholder="一行一个，支持单 IP、CIDR、通配符&#10;1.2.3.4&#10;127.*.*.*&#10;10.0.0.0/24"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="blockBatchVisible = false">取消</el-button>
        <el-button type="primary" :loading="blockBatchSubmitting" @click="submitBlockBatch">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed} from 'vue'
import { Search, ArrowDown } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDateInTimezone } from '@/utils/helpers'

const activeTab = ref('current')
const loading = ref(false)

// --- Current Blocked ---
const currentList = ref([])
const currentTotal = ref(0)
const currentSelections = ref([])
const currentQuery = reactive({ page: 1, pageSize: 10 })
const currentFilter = reactive({ type: 'ip', keyword: '' })
const blockBatchVisible = ref(false)
const blockBatchSubmitting = ref(false)
const blockBatchForm = reactive({ domain: '', text: '' })
const DISPLAY_TIMEZONE = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
const localTimeZoneLabel = computed(() => DISPLAY_TIMEZONE)

const normalizeSourceTime = (value) => {
  const text = String(value ?? '').trim()
  if (!text) return ''
  if (/[zZ]$|[+-]\d{2}:\d{2}$/.test(text)) return text
  // Backend returns wall-clock time without timezone; treat it as UTC+8 first.
  return text.replace(' ', 'T') + '+08:00'
}

const normalizeReleaseTime = (value) => {
  const text = String(value ?? '').trim()
  if (!text) return '-'
  if (text.toUpperCase() === 'PERMANENT') return '永久'
  return formatDateInTimezone(normalizeSourceTime(text), DISPLAY_TIMEZONE)
}

const normalizeBlockRowTime = (row) => {
  if (!row || typeof row !== 'object') return row
  return {
    ...row,
    block_time: formatDateInTimezone(normalizeSourceTime(row.block_time), DISPLAY_TIMEZONE),
    release_time: normalizeReleaseTime(row.release_time)
  }
}

const getCurrentRowKey = (row) => {
  const siteID = row?.site_id || ''
  const ip = row?.ip || ''
  return `${siteID}:${ip}`
}

const fetchCurrentList = async () => {
  loading.value = true
  try {
    const res = await request.get('/logs/block/current', {
      params: {
        ...currentQuery,
        type: currentFilter.type,
        keyword: currentFilter.keyword
      }
    })
    currentList.value = (res.data?.list || []).map(normalizeBlockRowTime)
    currentTotal.value = res.data?.total || 0
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

const handleCurrentSelectionChange = (rows) => {
  currentSelections.value = Array.isArray(rows) ? rows : []
}

const openBlockBatchDialog = () => {
  const domains = [...new Set(currentSelections.value.map(r => String(r?.domain || '').trim()).filter(Boolean))]
  const ips = [...new Set(currentSelections.value.map(r => String(r?.ip || '').trim()).filter(Boolean))]
  blockBatchForm.domain = domains.length === 1 ? domains[0] : (domains[0] || '')
  blockBatchForm.text = ips.join('\n')
  blockBatchVisible.value = true
}

const submitBlockBatch = async () => {
  const domain = String(blockBatchForm.domain || '').trim()
  const text = String(blockBatchForm.text || '').trim()
  if (!domain) {
    ElMessage.warning('请填写域名')
    return
  }
  if (!text) {
    ElMessage.warning('请填写至少一个 IP 或 IP 范围')
    return
  }
  blockBatchSubmitting.value = true
  try {
    await ElMessageBox.confirm(`确定将列表中的 IP 加入 ${domain} 的黑名单？`, '加入黑名单', { type: 'warning' })
    const res = await request.post('/logs/block/block_batch', { domain, text })
    const data = res.data || {}
    const added = Number(data.added || 0)
    const skipped = Number(data.skipped || 0)
    const invalid = Array.isArray(data.invalid) ? data.invalid : []
    let msg = `成功加入 ${added} 条`
    if (skipped > 0) msg += `，跳过重复 ${skipped} 条`
    if (invalid.length > 0) msg += `，无效 ${invalid.length} 条`
    ElMessage.success(msg)
    blockBatchVisible.value = false
  } catch (error) {
    if (error !== 'cancel') {
      console.error(error)
      ElMessage.error('加入黑名单失败')
    }
  } finally {
    blockBatchSubmitting.value = false
  }
}

const handleUnblockBatch = async () => {
  if (!currentSelections.value.length) {
    ElMessage.info('\u8bf7\u9009\u62e9\u9700\u8981\u89e3\u5c01\u7684\u8bb0\u5f55')
    return
  }
  loading.value = true
  try {
    await request.post('/logs/block/unblock_batch', {
      items: currentSelections.value.map(r => ({ ip: r.ip, domain: r.domain }))
    })
    ElMessage.success('\u6279\u91cf\u89e3\u5c01\u6210\u529f')
    await fetchCurrentList()
  } catch (error) {
    console.error(error)
    ElMessage.error('\u6279\u91cf\u89e3\u5c01\u5931\u8d25')
  } finally {
    loading.value = false
  }
}
const handleUnblockSite = async () => {
  if (!currentSelections.value.length) {
    ElMessage.info('\u8bf7\u9009\u62e9\u8981\u89e3\u5c01\u7684\u7f51\u7ad9')
    return
  }
  const ids = [...new Set(currentSelections.value.map(r => Number(r.site_id)).filter(id => id > 0))]
  if (!ids.length) {
    ElMessage.warning('\u9009\u4e2d\u8bb0\u5f55\u6ca1\u6709\u53ef\u7528\u7f51\u7ad9ID')
    return
  }
  loading.value = true
  try {
    await request.post('/logs/block/unblock_site', { site_ids: ids })
    ElMessage.success('\u7f51\u7ad9\u89e3\u5c01\u6210\u529f')
    await fetchCurrentList()
  } catch (error) {
    console.error(error)
    ElMessage.error('\u7f51\u7ad9\u89e3\u5c01\u5931\u8d25')
  } finally {
    loading.value = false
  }
}
const handleExportCurrent = () => {
  ElMessage.info('\u8bf7\u5148\u9009\u62e9\u8bb0\u5f55')
}
const handleUnblock = async row => {
  const ip = String(row?.ip || '').trim()
  if (!ip) {
    ElMessage.warning('IP \u4e0d\u53ef\u4e3a\u7a7a')
    return
  }
  loading.value = true
  try {
    await request.post('/logs/block/unblock_ip', { ip, domain: row?.domain })
    ElMessage.success(`\u89e3\u5c01\u6210\u529f IP: ${ip}`)
    await fetchCurrentList()
  } catch (error) {
    console.error(error)
    ElMessage.error('\u89e3\u5c01\u5931\u8d25')
  } finally {
    loading.value = false
  }
}

// --- Statistics ---
const statsType = ref('rank')
const statsList = ref([])
const statsTotal = ref(0)
const statsQuery = reactive({ page: 1, pageSize: 10 })

const fetchStatsList = async () => {
  loading.value = true
  try {
    const res = await request.get('/logs/block/stats', {
      params: { ...statsQuery }
    })
    statsList.value = res.data?.list || []
    statsTotal.value = res.data?.total || 0
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

// --- History ---
const historyList = ref([])
const historyTotal = ref(0)
const historyQuery = reactive({ page: 1, pageSize: 10 })
const historyFilter = reactive({ type: 'ip', keyword: '', dateRange: [] })

const historyFilterLabel = computed(() => {
  const map = { ip: 'IP', site_id: '\u7f51\u7ad9ID', time_range: '\u65f6\u95f4\u8303\u56f4' }
  return map[historyFilter.type] || 'IP'
})

const handleHistoryFilterCommand = command => {
  historyFilter.type = command
  historyFilter.keyword = ''
  historyFilter.dateRange = []
}

const fetchHistoryList = async () => {
  loading.value = true
  try {
    const res = await request.get('/logs/block/history', {
      params: {
        ...historyQuery,
        type: historyFilter.type,
        keyword: historyFilter.keyword,
        start_time: historyFilter.dateRange?.[0],
        end_time: historyFilter.dateRange?.[1]
      }
    })
    historyList.value = (res.data?.list || []).map(normalizeBlockRowTime)
    historyTotal.value = res.data?.total || 0
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}




const handleExportHistory = () => {
  ElMessage.info('\u8bf7\u5148\u9009\u62e9\u8bb0\u5f55')
}

const handleTabChange = name => {
  if (name === 'current') {
    fetchCurrentList()
    return
  }
  if (name === 'stats') {
    fetchStatsList()
    return
  }
  if (name === 'history') {
    fetchHistoryList()
  }
}

onMounted(() => {
  fetchCurrentList()
})
</script>

<style scoped>
.filter-container {
  margin-bottom: 20px;
}
.timezone-note {
  margin-right: 12px;
  color: #909399;
  font-size: 12px;
}
.filter-item {
  margin-right: 10px;
}
.filter-inline {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
.el-dropdown-link {
  cursor: pointer;
  color: var(--el-color-primary);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
</style>


