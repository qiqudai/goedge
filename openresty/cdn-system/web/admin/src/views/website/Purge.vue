<template>
  <div class="app-container">
    <el-tabs v-model="activeName">
      <el-tab-pane label="刷新预热" name="create">
        <el-form :model="form" label-width="120px" style="margin-top: 20px;">
          <el-form-item label="操作类型:">
            <el-radio-group v-model="form.type">
              <el-radio label="refresh_url">刷新URL</el-radio>
              <el-radio label="refresh_dir">刷新目录</el-radio>
              <el-radio label="preheat">预热</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="URL:">
            <el-input
              v-model="form.urls"
              type="textarea"
              :rows="10"
              placeholder="一行一条URL"
              style="width: 600px;"
            />
          </el-form-item>
          <div style="margin-left: 120px; color: #909399; margin-bottom: 20px;">
            每日限额0次, 今日剩余0次
          </div>
          <el-form-item>
            <el-button type="primary" @click="onSubmit" :loading="submitLoading">提交</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="操作记录" name="list">
        <div class="filter-container" style="margin-bottom: 20px;">
          <el-button type="primary" class="filter-item">重新提交</el-button>
          <el-select v-model="listQuery.type" placeholder="不限类型" clearable class="filter-item" style="width: 130px; margin-left: 10px;">
            <el-option label="刷新URL" value="refresh_url" />
            <el-option label="刷新目录" value="refresh_dir" />
            <el-option label="预热" value="preheat" />
          </el-select>
          <el-input v-model="listQuery.keyword" placeholder="URL或域名" style="width: 200px; margin-left: 10px;" class="filter-item" @keyup.enter="handleFilter" />
          <el-button class="filter-item" type="primary" icon="Search" @click="handleFilter" style="margin-left: 10px;" circle plain />
        </div>

        <el-table :data="list" v-loading="listLoading" border style="width: 100%">
          <el-table-column type="selection" width="55" />
          <el-table-column prop="id" label="JobId / TaskId" width="120" />
          <el-table-column prop="type" label="类型" width="120">
            <template #default="{row}">
              {{ typeMap[row.type] || row.type }}
            </template>
          </el-table-column>
          <el-table-column prop="data" label="URL">
             <template #default="{row}">
                 <div style="max-height: 100px; overflow-y: auto;">{{ row.data }}</div>
             </template>
          </el-table-column>
          <el-table-column prop="state" label="状态" width="100">
            <template #default="{row}">
              <el-tag :type="statusTypeMap[row.state]">{{ row.state }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="create_at" label="创建时间" width="180">
             <template #default="{row}">
                 {{ formatTime(row.create_at) }}
             </template>
          </el-table-column>
          <el-table-column label="操作" width="100">
             <template #default="{row}">
                <!-- Placeholder for action -->
                <el-button type="text">查看</el-button>
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
  'refresh_url': '刷新URL',
  'refresh_dir': '刷新目录',
  'preheat': '预热'
}

const statusTypeMap = {
  'waiting': 'info',
  'processing': 'warning',
  'done': 'success',
  'fail': 'danger'
}

// Format time simply strictly ISO string or use a library
const formatTime = (t) => {
    if (!t) return ''
    return t.replace('T', ' ').substring(0, 19)
}

const onSubmit = () => {
  if (!form.urls) {
    ElMessage.warning('请输入URL')
    return
  }
  submitLoading.value = true
  request.post('/tasks', form).then(() => {
    ElMessage.success('提交成功')
    form.urls = ''
    submitLoading.value = false
    activeName.value = 'list' // switch to list
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
    list.value = res.list || []
    total.value = res.total || 0
    listLoading.value = false
  }).catch(() => {
    listLoading.value = false
  })
}

const handleFilter = () => {
    listQuery.page = 1
    fetchList()
}

watch(activeName, (val) => {
  if (val === 'list') {
    fetchList()
  }
})

onMounted(() => {
    // Optionally fetch list if waiting on list tab
})
</script>
