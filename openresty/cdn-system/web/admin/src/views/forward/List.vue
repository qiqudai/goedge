<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="activeTopTab" class="custom-tabs" @tab-change="handleTopTab">
        <el-tab-pane label="转发列表" name="list">
          <ForwardTable
            :list="list"
            :total="total"
            :loading="listLoading"
            :selected-rows="selectedRows"
            @search="handleSearch"
            @create="handleCreate"
            @edit="handleEdit"
            @batch-edit="handleBatchEdit"
            @batch-action="handleBatchAction"
            @selection-change="(rows) => selectedRows = rows"
            @row-action="handleRowAction"
            @advanced="advancedVisible = true"
          />
        </el-tab-pane>
        <el-tab-pane label="默认设置" name="default" />
        <el-tab-pane label="实时监控" name="monitor" />
      </el-tabs>
    </el-card>

    <!-- Advanced Search Dialog -->
    <el-dialog v-model="advancedVisible" title="高级搜索" width="500px">
       <el-form label-width="100px">
          <el-form-item label="状态">
             <el-select v-model="advQuery.status" style="width: 100%;">
                <el-option label="正常" value="enabled" />
                <el-option label="停用" value="disabled" />
             </el-select>
          </el-form-item>
       </el-form>
       <template #footer>
          <el-button @click="advancedVisible = false">取消</el-button>
          <el-button type="primary" @click="applyAdvancedSearch">搜索</el-button>
       </template>
    </el-dialog>

    <ForwardEditDialog
      v-model="editVisible"
      :data="editData"
      @success="fetchList"
    />
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

import ForwardTable from './list/ForwardTable.vue'
import ForwardEditDialog from './list/ForwardEditDialog.vue'

const router = useRouter()
const activeTopTab = ref('list')

const list = ref([])
const total = ref(0)
const listLoading = ref(false)
const selectedRows = ref([])
const listQuery = reactive({ page: 1, pageSize: 10, keyword: '', searchField: 'listen' })

const editVisible = ref(false)
const editData = ref(null)

const advancedVisible = ref(false)
const advQuery = reactive({ status: '' })

const fetchList = async () => {
  listLoading.value = true
  try {
    const res = await request.get('/forwards', { params: { ...listQuery, ...advQuery } })
    list.value = res.list || []
    total.value = res.total || 0
  } finally {
    listLoading.value = false
  }
}

const handleTopTab = (name) => {
  const map = {
    list: '/forward/list',
    default: '/forward/default',
    monitor: '/forward/monitor'
  }
  const path = map[name]
  if (path && name !== 'list') router.push(path)
}

const handleSearch = (q) => { Object.assign(listQuery, q); fetchList() }

const handleCreate = () => {
    editData.value = null
    editVisible.value = true
}

const handleEdit = (row) => {
    editData.value = { ...row }
    editVisible.value = true
}

const handleBatchEdit = () => {
    // open batch edit logic
}

const handleBatchAction = async (action) => {
  const ids = selectedRows.value.map(r => r.id)
  await ElMessageBox.confirm(`确定执行批量${action}操作吗？`, '提示')
  await request.post('/forwards/batch_action', { action, ids })
  ElMessage.success('操作成功')
  fetchList()
}

const handleRowAction = (action, row) => {
    selectedRows.value = [row]
    handleBatchAction(action)
}

const applyAdvancedSearch = () => {
  advancedVisible.value = false
  fetchList()
}

onMounted(() => {
  fetchList()
})
</script>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
</style>
