<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" @tab-click="handleTabClick">
      <el-tab-pane label="当前封禁" name="current">
        <div class="filter-container">
          <el-button type="primary" class="filter-item" @click="handleUnblockBatch">批量解封</el-button>
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
          v-model:current-page="currentQuery.page"
          v-model:page-size="currentQuery.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, prev, pager, next, sizes, jumper"
          :total="currentTotal"
          persist-key="current"
          @size-change="fetchCurrentList"
          @current-change="fetchCurrentList"
        >
          <el-table-column type="selection" width="55" />
          <el-table-column prop="site_id" label="网站ID" width="100" />
          <el-table-column prop="domain" label="域名" />
          <el-table-column prop="ip" label="IP" />
          <el-table-column prop="location" label="域名" />
          <el-table-column prop="filter" label="规则" />
          <el-table-column prop="block_time" label="封禁时间" />
          <el-table-column prop="release_time" label="封禁时间" />
          <el-table-column label="操作" width="100">
            <template #default="scope">
              <el-button type="text" size="small" @click="handleUnblock(scope.row)">解封</el-button>
            </template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>

      <el-tab-pane label="统计" name="stats">
        <div class="filter-container">
          <el-radio-group v-model="statsType" style="margin-bottom: 20px;">
            <el-radio-button label="rank">排行</el-radio-button>
          </el-radio-group>
        </div>

        <AppTable
          :data="statsList"
          :loading="loading"
          border
          style="width: 100%"
          v-model:current-page="statsQuery.page"
          v-model:page-size="statsQuery.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, prev, pager, next, sizes, jumper"
          :total="statsTotal"
          persist-key="stats"
          @size-change="fetchStatsList"
          @current-change="fetchStatsList"
        >
          <el-table-column prop="site_id" label="网站ID" />
          <el-table-column prop="count" label="封禁时间" />
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
              range-separator="?"
              start-placeholder="????"
              end-placeholder="????"
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
          :page-sizes="[10, 20, 50, 100]"
          layout="total, prev, pager, next, sizes, jumper"
          :total="historyTotal"
          persist-key="history"
          @size-change="fetchHistoryList"
          @current-change="fetchHistoryList"
        >
          <el-table-column prop="site_id" label="网站ID" width="100" />
          <el-table-column prop="domain" label="域名" />
          <el-table-column prop="ip" label="IP" />
          <el-table-column prop="location" label="域名" />
          <el-table-column prop="filter" label="规则" />
          <el-table-column prop="block_time" label="封禁时间" />
          <el-table-column prop="is_manual" label="解封方式">
            <template #default="scope">{{ scope.row.is_manual ? '\u624b\u52a8' : '\u81ea\u52a8' }}</template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed} from 'vue'
import { Search, ArrowDown } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const activeTab = ref('current')
const loading = ref(false)

// --- Current Blocked ---
const currentList = ref([])
const currentTotal = ref(0)
const currentQuery = reactive({ page: 1, pageSize: 10 })
const currentFilter = reactive({ type: 'ip', keyword: '' })

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
    currentList.value = res.data?.list || []
    currentTotal.value = res.data?.total || 0
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

const handleUnblockBatch = () => {
  ElMessage.info('\u8bf7\u9009\u62e9\u9700\u8981\u89e3\u5c01\u7684\u8bb0\u5f55')
}
const handleUnblockSite = () => {
  ElMessage.info('\u8bf7\u9009\u62e9\u8981\u89e3\u5c01\u7684\u7f51\u7ad9')
}
const handleExportCurrent = () => {
  ElMessage.info('\u8bf7\u5148\u9009\u62e9\u8bb0\u5f55')
}
const handleUnblock = row => {
  ElMessage.success(`\u89e3\u5c01\u6210\u529f IP: ${row.ip}`)
}

// --- Statistics ---
const statsType = ref('rank')
const statsList = ref([])
const statsTotal = ref(0)
const statsQuery = reactive({ page: 1, pageSize: 10 })

const fetchStatsList = async () => {
  if (activeTab.value !== 'stats') return
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
  if (activeTab.value !== 'history') return
  loading.value = true
  try {
    const res = await request.get('/logs/block/history', {
      params: {
        ...historyQuery,
        type: historyFilter.type,
        keyword: historyFilter.keyword
      }
    })
    historyList.value = res.data?.list || []
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

const handleTabClick = tab => {
  if (tab.props.name === 'current') {
    fetchCurrentList()
  } else if (tab.props.name === 'stats') {
    fetchStatsList()
  } else if (tab.props.name === 'history') {
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


