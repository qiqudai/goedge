<template>
  <div class="app-container">
    <el-tabs v-model="activeName">
      <el-tab-pane label="刷新预热" name="create">
        <el-form :model="form" label-width="120px" style="margin-top: 20px;">
          <el-form-item label="类型">
            <el-radio-group v-model="form.type">
              <el-radio label="refresh_url">刷新URL</el-radio>
              <el-radio label="refresh_dir">刷新目录</el-radio>
              <el-radio label="preheat">预热</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="URL">
            <el-input
              v-model="form.urls"
              type="textarea"
              :rows="10"
              placeholder="请输入URL"
              style="width: 600px;"
            />
          </el-form-item>
          <div style="margin-left: 120px; color: #909399; margin-bottom: 20px;">
            最多100条 每行1条
          </div>
          <el-form-item>
            <el-button type="primary" @click="onSubmit" :loading="submitLoading">提交</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="任务列表" name="list">
        <div class="filter-container" style="margin-bottom: 20px;">
          <el-button type="primary" class="filter-item">批量删除</el-button>
          <el-select v-model="listQuery.type" placeholder="类型" clearable class="filter-item" style="width: 130px; margin-left: 10px;">
            <el-option label="刷新URL" value="refresh_url" />
            <el-option label="刷新目录" value="refresh_dir" />
            <el-option label="预热" value="preheat" />
          </el-select>
          <el-input v-model="listQuery.keyword" placeholder="URL模糊搜索" style="width: 200px; margin-left: 10px;" class="filter-item" @keyup.enter="handleFilter" />
          <el-button class="filter-item" type="primary" :icon="Search" @click="handleFilter" style="margin-left: 10px;" circle plain />
        </div>

        <el-table :data="list" v-loading="listLoading" border style="width: 100%">
          <el-table-column type="selection" width="55" />
          <el-table-column prop="id" label="JobId / TaskId" width="120" />
          <el-table-column prop="type" label="类型" width="120">
            <template #default="{ row }">
              {{ typeMap[row.type] || row.type }}
            </template>
          </el-table-column>
          <el-table-column prop="data" label="URL">
            <template #default="{ row }">
              <div style="max-height: 100px; overflow-y: auto;">{{ row.data }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="state" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="statusTypeMap[row.state]">{{ row.state }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="create_at" label="创建时间" width="180">
            <template #default="{ row }">{{ formatTime(row.create_at) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default>
              <el-button type="text">详情</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-container" style="margin-top: 20px;">
          <el-pagination
            v-model:current-page="listQuery.page"
            v-model:page-size="listQuery.limit"
            :total="total"
            layout="total, prev, pager, next, sizes"
            @current-change="fetchList"
            @size-change="fetchList"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'

const activeName = ref('create')
const submitLoading = ref(false)

const form = reactive({
  type: 'refresh_url',
  urls: ''
})

const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const listQuery = reactive({
  page: 1,
  limit: 10,
  keyword: '',
  type: ''
})

const typeMap = {
  refresh_url: '\u5237\u65b0URL',
  refresh_dir: '\u5237\u65b0\u76ee\u5f55',
  preheat: '\u9884\u70ed'
}

const statusTypeMap = {
  waiting: 'info',
  processing: 'warning',
  done: 'success',
  fail: 'danger'
}

const formatTime = t => {
  if (!t) return ''
  return String(t).replace('T', ' ').substring(0, 19)
}

const onSubmit = () => {
  if (!form.urls) {
    ElMessage.warning('\u8bf7\u8f93\u5165URL')
    return
  }
  submitLoading.value = true
  request.post('/tasks', form).then(() => {
    ElMessage.success('\u63d0\u4ea4\u6210\u529f')
    form.urls = ''
    submitLoading.value = false
    activeName.value = 'list'
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

watch(activeName, val => {
  if (val === 'list') {
    fetchList()
  }
})

onMounted(() => {
  // keep lazy loading
})
</script>
