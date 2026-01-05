<template>
  <div class="app-container">
    <div class="filter-container">
      <el-select v-model="filters.type" placeholder="关键词" clearable style="width: 160px;">
        <el-option label="刷新URL" value="refresh_url" />
        <el-option label="刷新目录" value="refresh_dir" />
        <el-option label="预热" value="preheat" />
      </el-select>
      <el-input v-model="filters.keyword" placeholder="关键词" style="width: 240px;" />
      <el-button type="primary" :loading="loading" @click="loadList">查询</el-button>
    </div>

    <AppTable
      :data="list"
      :loading="loading"
      border
      row-key="id"
      style="width: 100%;"
      v-model:current-page="filters.page"
      v-model:page-size="filters.pageSize"

      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="loadList"
      @current-change="loadList"
    >
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="pry" label="优先级" width="80" />
      <el-table-column prop="name" label="名称" width="120" show-overflow-tooltip />
      <el-table-column prop="type" label="类型" width="120">
        <template #default="scope">
          <el-tag>{{ formatType(scope.row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="pid" label="资源ID" width="100" />
      <el-table-column prop="depend" label="依赖" width="120" show-overflow-tooltip />
      <el-table-column prop="start_at" label="开始时间" width="160">
        <template #default="scope">
          {{ formatDate(scope.row.start_at) }}
        </template>
      </el-table-column>
      <el-table-column label="耗时" width="100">
        <template #default="scope">
           {{ formatDuration(scope.row) }}
        </template>
      </el-table-column>
      <el-table-column prop="state" label="状态" width="100">
        <template #default="scope">
          <el-tag :type="getStateType(scope.row.state)">
            {{ formatState(scope.row.state) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="err_times" label="失败次数" width="80" />
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="scope">
          <el-button link type="primary" @click="handleResubmit(scope.row)">重试</el-button>
        </template>
      </el-table-column>
    </AppTable>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted} from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const total = ref(0)
const loading = ref(false)

const filters = reactive({
  type: '',
  keyword: '',
  page: 1,
  pageSize: 20
})

const loadList = () => {
  if (loading.value) return
  loading.value = true
  request.get('/tasks', { params: { ...filters } }).then(res => {
      list.value = res.data?.list || res.list || []
      total.value = res.data?.total || res.total || 0
    })
    .finally(() => {
      loading.value = false
    })
}

const handleResubmit = (row) => {
    ElMessageBox.confirm('确定要重新提交该任务吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
    }).then(() => {
        request.post(`/tasks/${row.id}/resubmit`).then(() => {
            ElMessage.success('已重新提交')
            loadList()
        })
    })
}

// Helpers
const formatType = (val) => {
    const map = {
        'refresh_url': '刷新URL',
        'refresh_dir': '刷新目录',
        'preheat': '预热'
    }
    return map[val] || val
}

const formatState = (val) => {
    const map = {
        'waiting': '等待中',
        'running': '执行中',
        'done': '完成',
        'fail': '失败'
    }
    return map[val] || val
}

const getStateType = (val) => {
    const map = {
        'waiting': 'info',
        'running': 'primary',
        'done': 'success',
        'fail': 'danger'
    }
    return map[val] || ''
}

const formatDate = (val) => {
    if (!val || val.startsWith('0001')) return '-'
    return val.replace('T', ' ').substring(0, 19)
}

const formatDuration = (row) => {
    if (!row.start_at || row.start_at.startsWith('0001')) return '-'
    const start = new Date(row.start_at).getTime()
    const end = row.end_at && !row.end_at.startsWith('0001') ? new Date(row.end_at).getTime() : Date.now()
    if (end < start) return '-'
    const diff = end - start
    if (diff < 1000) return diff + 'ms'
    return (diff / 1000).toFixed(1) + 's'
}

onMounted(() => loadList())

</script>

<style scoped>
.filter-container {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
  margin-bottom: 16px;
}
.pagination-container {
  margin-top: 16px;
  text-align: right;
}
</style>



