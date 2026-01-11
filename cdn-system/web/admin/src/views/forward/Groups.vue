<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <template #header>
        <div class="card-header">
           <span class="title">分组设置</span>
        </div>
      </template>

      <div class="filter-container">
        <div class="filter-left">
          <el-button type="primary" @click="openCreate">添加分组</el-button>
          <el-button type="danger" :disabled="!selectedRows.length" @click="batchDelete">删除分组</el-button>
        </div>
        <div class="filter-right">
          <el-input
            v-model="keyword"
            placeholder="搜索分组名称"
            style="width: 200px;"
            @keyup.enter="fetchGroups"
          >
            <template #suffix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button type="primary" @click="fetchGroups">查询</el-button>
        </div>
      </div>

      <AppTable
        v-loading="loading"
        :data="groups"
        border
        fit
        highlight-current-row
        persist-key="forward-group-list"
        style="width: 100%;"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="50" align="center" />
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="name" label="分组名称" min-width="200" />
        <el-table-column prop="remark" label="备注" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="160" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="removeGroup(row)">删除</el-button>
          </template>
        </el-table-column>
      </AppTable>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px">
      <el-form :model="form" label-width="80px" style="padding-top: 10px;">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="请输入分组名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" placeholder="请输入备注信息" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import request from '@/utils/request'

const groups = ref([])
const loading = ref(false)
const keyword = ref('')
const dialogVisible = ref(false)
const editingId = ref(0)
const selectedRows = ref([])
const form = reactive({
  name: '',
  remark: ''
})

const dialogTitle = computed(() => (editingId.value ? '编辑分组' : '添加分组'))

const fetchGroups = () => {
  loading.value = true
  request.get('/forward_groups', { params: { keyword: keyword.value } }).then(res => {
    groups.value = res.data?.list || []
    loading.value = false
  }).catch(() => {
    loading.value = false
  })
}

const openCreate = () => {
  editingId.value = 0
  form.name = ''
  form.remark = ''
  dialogVisible.value = true
}

const openEdit = row => {
  editingId.value = row.id
  form.name = row.name || ''
  form.remark = row.remark || ''
  dialogVisible.value = true
}

const submitForm = () => {
  const payload = { id: editingId.value, name: form.name, remark: form.remark }
  if (editingId.value) {
    request.put('/forward_groups', payload).then(() => {
      ElMessage.success('更新成功')
      dialogVisible.value = false
      fetchGroups()
    })
  } else {
    request.post('/forward_groups', payload).then(() => {
      ElMessage.success('创建成功')
      dialogVisible.value = false
      fetchGroups()
    })
  }
}

const removeGroup = row => {
  ElMessageBox.confirm('确认删除该分组?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.delete('/forward_groups', { data: { id: row.id } }).then(() => {
      ElMessage.success('删除成功')
      fetchGroups()
    })
  })
}

const handleSelectionChange = rows => {
  selectedRows.value = rows
}

const batchDelete = () => {
  if (selectedRows.value.length === 0) return
  ElMessageBox.confirm(`确认删除选中的 ${selectedRows.value.length} 个分组?`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      // Use Promise.all for batch deletion since generic batch API might not exist
      const promises = selectedRows.value.map(row => request.delete('/forward_groups', { data: { id: row.id } }))
      await Promise.all(promises)
      ElMessage.success('批量删除成功')
      fetchGroups()
      selectedRows.value = []
    } catch (err) {
      // detailed error handled by request interceptor usually
    }
  })
}

onMounted(fetchGroups)
</script>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
.card-header { display: flex; gap: 12px; align-items: center; }
.card-header .title { font-size: 16px; font-weight: 600; }
.filter-container { margin-bottom: 20px; display: flex; gap: 12px; align-items: center; }
.filter-left { display: flex; gap: 10px; }
.filter-right { display: flex; gap: 10px; }
</style>


