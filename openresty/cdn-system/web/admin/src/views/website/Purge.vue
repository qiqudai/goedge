<template>
  <div class="app-container">
    <el-tabs v-model="activeName">
      <el-tab-pane label="刷新预热" name="create">
        <el-form :model="form" label-width="100px" class="purge-form">
          <el-form-item label="操作类型">
            <el-radio-group v-model="form.type" @change="loadUsage">
              <el-radio label="refresh_url">刷新URL</el-radio>
              <el-radio label="refresh_dir">刷新目录</el-radio>
              <el-radio label="preheat">预热</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="URL">
            <el-input
              v-model="form.urls"
              type="textarea"
              :rows="12"
              placeholder="URL或域名"
              class="url-textarea"
            />
            <div class="limit-tip">
              {{ limitText }}
              <el-button link type="primary" size="small" @click="loadUsage">刷新</el-button>
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="submitLoading" @click="onSubmit">提交</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="操作记录" name="list">
        <div class="filter-container">
          <el-button type="primary" :disabled="!selectedRows.length" @click="handleResubmitBatch">重新提交</el-button>
          <el-select v-model="listQuery.type" placeholder="URL或域名" clearable class="filter-item" style="width: 140px;">
            <el-option label="刷新URL" value="refresh_url" />
            <el-option label="刷新目录" value="refresh_dir" />
            <el-option label="预热" value="preheat" />
          </el-select>
          <el-input
            v-model="listQuery.keyword"
            placeholder="URL或域名"
            class="filter-item"
            style="width: 220px;"
            @keyup.enter="handleFilter"
          />
          <el-button type="primary" class="filter-item" @click="handleFilter">查询</el-button>
        </div>

        <AppTable
          :data="list"
          :loading="listLoading"
          border
          style="width: 100%"
          @selection-change="handleSelectionChange"
          v-model:current-page="listQuery.page"
          v-model:page-size="listQuery.limit"
          :total="total"
          layout="total, prev, pager, next, sizes"
          @current-change="fetchList"
          @size-change="fetchList"
        >
          <el-table-column type="selection" width="55" />
          <el-table-column prop="id" label="JobId / TaskId" width="130" />
          <el-table-column prop="type" label="类型" width="120">
            <template #default="{ row }">
              {{ typeMap[row.type] || row.type }}
            </template>
          </el-table-column>
          <el-table-column prop="data" label="URL">
            <template #default="{ row }">
              <div class="url-cell">{{ row.data }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="state" label="状态" width="120">
            <template #default="{ row }">
              <el-tag :type="statusTypeMap[row.state] || 'info'">
                {{ statusTextMap[row.state] || row.state }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="create_at" label="创建时间" width="180">
            <template #default="{ row }">{{ formatTime(row.create_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="120" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="normal" @click="handleResubmit(row)">重新提交</el-button>
            </template>
          </el-table-column>
        </AppTable>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeName = ref('create')
const submitLoading = ref(false)
const listLoading = ref(false)
const list = ref([])
const total = ref(0)
const selectedRows = ref([])

const form = reactive({
  type: 'refresh_url',
  urls: ''
})

const listQuery = reactive({
  page: 1,
  limit: 10,
  keyword: '',
  type: ''
})


const usage = reactive({
  limits: { refresh_url: 0, refresh_dir: 0, preheat: 0 },
  remaining: { refresh_url: 0, refresh_dir: 0, preheat: 0 }
})

const typeMap = {
  refresh_url: '刷新URL',
  refresh_dir: '刷新目录',
  preheat: '预热'
}

const statusTypeMap = {
  waiting: 'info',
  running: 'warning',
  done: 'success',
  fail: 'danger'
}

const statusTextMap = {
  waiting: '等待',
  running: '处理中',
  done: '完成',
  fail: '失败'
}

const limitText = computed(() => {
  const limit = usage.limits[form.type] || 0
  const remaining = usage.remaining[form.type] || 0
  return `每日限额${limit}次，今日剩余${remaining}次`
})

const formatTime = t => {
  if (!t) return ''
  return String(t).replace('T', ' ').substring(0, 19)
}

const loadUsage = () => {
  request.get('/tasks/usage').then(res => {
    const data = res.data || {}
    usage.limits = data.limits || usage.limits
    usage.remaining = data.remaining || usage.remaining
  })
}

const onSubmit = () => {
  if (submitLoading.value) return
  if (!form.urls.trim()) {
    ElMessage.warning('请输入URL')
    return
  }
  submitLoading.value = true
  request.post('/tasks', form).then(() => {
    ElMessage.success('提交成功，请到操作记录里查看进度')
    form.urls = ''
    submitLoading.value = false
    activeName.value = 'list'
    loadUsage()
    fetchList()
    }).catch(() => {
      submitLoading.value = false
    })
}

const fetchList = () => {
  listLoading.value = true
  request.get('/tasks', {
    params: {
      page: listQuery.page,
      pageSize: listQuery.limit,
      keyword: listQuery.keyword,
      type: listQuery.type
    }
  }).then(res => {
    list.value = res.data?.list || res.list || []
    total.value = res.data?.total || res.total || 0
    listLoading.value = false
  }).catch(() => {
    listLoading.value = false
  })
}

const handleFilter = () => {
  listQuery.page = 1
  fetchList()
}

const handleSelectionChange = rows => {
  selectedRows.value = rows
}

const handleResubmit = row => {
  ElMessageBox.confirm('确认重新提交该任�?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.post(`/tasks/${row.id}/resubmit`).then(() => {
      ElMessage.success('重新提交成功')
      loadUsage()
      fetchList()
      }).catch(() => {
      })
    })
  }

const handleResubmitBatch = () => {
  if (!selectedRows.value.length) return
  ElMessageBox.confirm('确认重新提交选中任务?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    Promise.all(selectedRows.value.map(row => request.post(`/tasks/${row.id}/resubmit`)))
      .then(() => {
        ElMessage.success('重新提交成功')
        loadUsage()
        fetchList()
        }).catch(() => {
        })
    })
  }

watch(activeName, val => {
  if (val === 'list') {
    fetchList()
  }
})

onMounted(() => {
  loadUsage()
})
</script>

<style scoped>
.purge-form {
  margin-top: 20px;
  max-width: 720px;
}
.url-textarea {
  width: 100%;
}
.limit-tip {
  color: #909399;
  margin-top: 6px;
  font-size: 12px;
}
.filter-container {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.filter-item {
  margin-left: 8px;
}
.url-cell {
  max-height: 100px;
  overflow-y: auto;
  white-space: pre-wrap;
}
.pagination-container {
  margin-top: 20px;
  text-align: right;
}
</style>


