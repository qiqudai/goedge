<template>
  <div class="app-container">
    <!-- Filter Header -->
    <div class="filter-container">
      <div class="filter-left">
        <el-radio-group v-model="dateRangeType" @change="handleDateRangeChange" size="small">
          <el-radio-button value="today">今天</el-radio-button>
          <el-radio-button value="week">最近7天</el-radio-button>
          <el-radio-button value="month">最近30天</el-radio-button>
          <el-radio-button value="custom">自定义</el-radio-button>
        </el-radio-group>
        <el-date-picker
          v-if="dateRangeType === 'custom'"
          v-model="customDateRange"
          type="datetimerange"
          range-separator="至"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          size="small"
          style="margin-left: 10px; width: 340px;"
          @change="handleCustomDateChange"
        />
      </div>
      <div class="filter-right">
        <el-input 
          v-model="query.keyword" 
          :placeholder="searchPlaceholder" 
          style="width: 300px;" 
          size="small" 
          @keyup.enter="fetchData"
          clearable 
          @clear="fetchData"
        >
          <template #append>
            <el-button :icon="Search" @click="fetchData" />
          </template>
        </el-input>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="log-tabs" @tab-change="handleTabChange">
      <!-- 1. Login Log -->
      <!-- Columns: User ID, IP, Location, Time, Status -->
      <el-tab-pane label="登录日志" name="login">
        <AppTable 
          :data="list" 
          v-loading="loading" 
          border 
          fit 
          highlight-current-row 
          persist-key="log-login"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          v-model:current-page="query.page"
          v-model:page-size="query.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          @size-change="fetchData"
          @current-change="fetchData"
        >
          <el-table-column prop="user_id" label="用户ID" width="100">
             <template #default="{row}">
               {{ row.user_id || row.uid }}
               <span v-if="row.username" class="text-gray">({{ row.username }})</span>
             </template>
          </el-table-column>
          <el-table-column prop="ip" label="IP地址" width="140" />
          <el-table-column prop="region" label="地理位置" min-width="120" />
          <el-table-column prop="created_at" label="登录时间" width="160">
             <template #default="{row}">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="登录状态" width="100" align="center">
            <template #default="{row}">
               <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '成功' : '失败' }}</el-tag>
            </template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>

      <!-- 2. Operation Log -->
      <!-- Columns: User ID, Type, Object, Action, Content, IP, Location, Time -->
      <el-tab-pane label="操作日志" name="operation">
        <AppTable 
          :data="list" 
          v-loading="loading" 
          border 
          fit 
          highlight-current-row 
          persist-key="log-operation"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          v-model:current-page="query.page"
          v-model:page-size="query.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          @size-change="fetchData"
          @current-change="fetchData"
        >
          <el-table-column prop="user_id" label="用户ID" width="100">
             <template #default="{row}">{{ row.user_id || row.uid }}</template>
          </el-table-column>
          <el-table-column prop="type" label="类别" width="100" />
          <el-table-column prop="object" label="对象" width="120" show-overflow-tooltip />
          <el-table-column prop="action" label="动作" width="100" />
          <el-table-column prop="content" label="变更内容" min-width="200" show-overflow-tooltip />
          <el-table-column prop="ip" label="IP地址" width="130" />
          <el-table-column prop="region" label="地理位置" width="120" />
          <el-table-column prop="created_at" label="操作时间" width="160">
             <template #default="{row}">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>

      <!-- 3. Backup Log -->
      <!-- Columns: Backup Time, Finished Time, Status, Result -->
      <el-tab-pane label="备份日志" name="backup">
        <AppTable 
          :data="list" 
          v-loading="loading" 
          border 
          fit 
          highlight-current-row 
          persist-key="log-backup"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          v-model:current-page="query.page"
          v-model:page-size="query.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          @size-change="fetchData"
          @current-change="fetchData"
        >
          <el-table-column prop="created_at" label="备份时间" width="180">
             <template #default="{row}">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="finished_at" label="完成时间" width="180">
             <template #default="{row}">{{ formatTime(row.finished_at) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{row}">
               <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '成功' : '失败' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="result" label="结果" min-width="200" show-overflow-tooltip />
        </AppTable>
      </el-tab-pane>

      <!-- 4. Mail Log (Sending Log) -->
      <!-- Columns: User ID, Msg ID, Title, Medium, Fails, Status, Reason, Time -->
      <el-tab-pane label="发信日志" name="mail">
        <AppTable 
          :data="list" 
          v-loading="loading" 
          border 
          fit 
          highlight-current-row 
          persist-key="log-mail"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          v-model:current-page="query.page"
          v-model:page-size="query.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          @size-change="fetchData"
          @current-change="fetchData"
        >
          <el-table-column prop="user_id" label="用户ID" width="100">
             <template #default="{row}">{{ row.user_id || row.uid }}</template>
          </el-table-column>
          <el-table-column prop="message_id" label="消息ID" width="100" show-overflow-tooltip />
          <el-table-column prop="subject" label="标题" min-width="150" show-overflow-tooltip />
          <el-table-column prop="medium" label="媒介" width="100">
             <template #default="{row}">
               <el-tag size="small" type="info">{{ row.medium || 'Email' }}</el-tag>
             </template>
          </el-table-column>
          <el-table-column prop="fails" label="失败次数" width="90" align="center" />
          <el-table-column label="状态" width="100" align="center">
            <template #default="{row}">
               <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '成功' : '失败' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="150" show-overflow-tooltip />
          <el-table-column prop="created_at" label="发送时间" width="160">
             <template #default="{row}">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { Search } from '@element-plus/icons-vue'
import request from '@/utils/request'

const activeTab = ref('login')
const list = ref([])
const total = ref(0)
const loading = ref(false)
const dateRangeType = ref('today')
const customDateRange = ref([])

const query = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  start_time: 0,
  end_time: 0
})

const searchPlaceholder = computed(() => {
  switch (activeTab.value) {
    case 'login': return '搜索 IP / 用户ID / 地理位置 / 状态'
    case 'operation': return '搜索 IP / 用户ID / 地理位置 / 类别 / 对象 / 动作'
    case 'backup': return '搜索 状态 / 结果'
    case 'mail': return '搜索 标题 / 用户ID / 消息ID / 状态 / 原因'
    default: return '关键词搜索'
  }
})

const handleTabChange = () => {
  query.page = 1
  query.keyword = '' // Optional: clear keyword on tab change? User didn't specify, but usually safer.
  fetchData()
}

const handleDateRangeChange = (val) => {
  const now = new Date()
  let start = new Date()
  
  if (val === 'today') {
    start.setHours(0, 0, 0, 0)
  } else if (val === 'week') {
    start.setTime(now.getTime() - 3600 * 1000 * 24 * 7)
  } else if (val === 'month') {
    start.setTime(now.getTime() - 3600 * 1000 * 24 * 30)
  } else if (val === 'custom') {
    // wait for date picker
    return
  }
  
  query.start_time = Math.floor(start.getTime() / 1000)
  query.end_time = Math.floor(now.getTime() / 1000)
  fetchData()
}

const handleCustomDateChange = (val) => {
  if (val && val.length === 2) {
    query.start_time = Math.floor(val[0].getTime() / 1000)
    query.end_time = Math.floor(val[1].getTime() / 1000)
    fetchData()
  }
}

const fetchData = async () => {
  loading.value = true
  let url = ''
  switch (activeTab.value) {
    case 'login': url = '/logs/login'; break
    case 'operation': url = '/logs/operation'; break
    case 'backup': url = '/logs/backup'; break
    case 'mail': url = '/logs/mail'; break
  }
  
  try {
    const { data } = await request.get(url, { params: query })
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const formatTime = (ts) => {
  if (!ts) return '-'
  const date = new Date(ts * 1000)
  return date.toLocaleString()
}

onMounted(() => {
  handleDateRangeChange('today') // Init with today
})
</script>

<style scoped>
.filter-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  background-color: #fff;
  padding: 10px;
  border-radius: 4px;
}
.filter-left, .filter-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.pagination-container {
  margin-top: 20px;
  text-align: right;
  padding: 10px;
  background: #fff;
}
.text-gray {
  color: #909399;
  font-size: 0.9em;
  margin-left: 5px;
}
</style>
