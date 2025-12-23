<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" @tab-click="handleTabClick">
      <el-tab-pane label="????" name="current">
        <div class="filter-container">
          <el-button type="primary" class="filter-item" @click="handleUnblockBatch">??</el-button>
          <el-button class="filter-item" @click="handleUnblockSite">????</el-button>
          <el-button class="filter-item" @click="handleExportCurrent">?????</el-button>

          <div class="filter-item filter-inline">
            <el-select v-model="currentFilter.type" style="width: 120px" placeholder="??">
              <el-option label="IP??" value="ip" />
              <el-option label="??ID" value="site_id" />
            </el-select>
            <el-input
              v-model="currentFilter.keyword"
              placeholder="??????"
              style="width: 200px;"
              class="filter-item"
              @keyup.enter="fetchCurrentList"
            />
            <el-button class="filter-item" type="primary" :icon="Search" @click="fetchCurrentList" />
          </div>
        </div>

        <el-table :data="currentList" border style="width: 100%" v-loading="loading">
          <el-table-column type="selection" width="55" />
          <el-table-column prop="site_id" label="??ID" width="100" />
          <el-table-column prop="domain" label="??" />
          <el-table-column prop="ip" label="IP" />
          <el-table-column prop="location" label="??" />
          <el-table-column prop="filter" label="???" />
          <el-table-column prop="block_time" label="????" />
          <el-table-column prop="release_time" label="????" />
          <el-table-column label="??" width="100">
            <template #default="scope">
              <el-button type="text" size="small" @click="handleUnblock(scope.row)">??</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-container">
          <el-pagination
            v-model:current-page="currentQuery.page"
            v-model:page-size="currentQuery.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, prev, pager, next, sizes, jumper"
            :total="currentTotal"
            @size-change="fetchCurrentList"
            @current-change="fetchCurrentList"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="????" name="stats">
        <div class="filter-container">
          <el-radio-group v-model="statsType" style="margin-bottom: 20px;">
            <el-radio-button label="rank">??</el-radio-button>
          </el-radio-group>
        </div>

        <el-table :data="statsList" border style="width: 100%" v-loading="loading">
          <el-table-column prop="site_id" label="??ID" />
          <el-table-column prop="count" label="????" />
        </el-table>

        <div class="pagination-container">
          <el-pagination
            v-model:current-page="statsQuery.page"
            v-model:page-size="statsQuery.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, prev, pager, next, sizes, jumper"
            :total="statsTotal"
            @size-change="fetchStatsList"
            @current-change="fetchStatsList"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="????" name="history">
        <div class="filter-container">
          <el-button class="filter-item" @click="handleExportHistory">?????</el-button>

          <div class="filter-item filter-inline">
            <el-dropdown trigger="click" @command="handleHistoryFilterCommand" style="margin-right: 10px;">
              <span class="el-dropdown-link">
                {{ historyFilterLabel }}
                <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="ip">IP</el-dropdown-item>
                  <el-dropdown-item command="site_id">??ID</el-dropdown-item>
                  <el-dropdown-item command="time_range">????</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-input
              v-if="historyFilter.type !== 'time_range'"
              v-model="historyFilter.keyword"
              placeholder="??????"
              style="width: 200px;"
              class="filter-item"
              @keyup.enter="fetchHistoryList"
            />
            <el-date-picker
              v-else
              v-model="historyFilter.dateRange"
              type="daterange"
              range-separator="?"
              start-placeholder="????"
              end-placeholder="????"
              style="width: 250px"
              class="filter-item"
            />
            <el-button class="filter-item" type="primary" :icon="Search" @click="fetchHistoryList" />
          </div>
        </div>

        <el-table :data="historyList" border style="width: 100%" v-loading="loading">
          <el-table-column prop="site_id" label="??ID" width="100" />
          <el-table-column prop="domain" label="??" />
          <el-table-column prop="ip" label="IP" />
          <el-table-column prop="location" label="??" />
          <el-table-column prop="filter" label="???" />
          <el-table-column prop="block_time" label="????" />
          <el-table-column prop="is_manual" label="????">
            <template #default="scope">{{ scope.row.is_manual ? '?' : '?' }}</template>
          </el-table-column>
        </el-table>

        <div class="pagination-container">
          <el-pagination
            v-model:current-page="historyQuery.page"
            v-model:page-size="historyQuery.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, prev, pager, next, sizes, jumper"
            :total="historyTotal"
            @size-change="fetchHistoryList"
            @current-change="fetchHistoryList"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
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
  ElMessage.info('?????????')
}
const handleUnblockSite = () => {
  ElMessage.info('?????????')
}
const handleExportCurrent = () => {
  ElMessage.info('???????')
}
const handleUnblock = row => {
  ElMessage.success(`????? IP: ${row.ip}`)
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
  const map = { ip: 'IP', site_id: '??ID', time_range: '????' }
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
  ElMessage.info('???????')
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
