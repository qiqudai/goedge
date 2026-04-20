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
          <el-select
            v-if="isAdmin"
            v-model="selectedUserId"
            placeholder="请选择用户"
            style="width: 280px;"
            filterable
            clearable
            :loading="userLoading"
            @visible-change="handleUserDropdown"
            @change="handleUserChange"
          >
            <el-option
              v-for="u in users"
              :key="u.id"
              :label="`${u.name} (${u.username || u.email || u.id})`"
              :value="u.id"
            />
          </el-select>
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
        <el-form-item v-if="isAdmin" label="用户" required>
          <el-select
            v-model="form.user_id"
            placeholder="请选择用户"
            style="width: 100%;"
            filterable
            clearable
            :loading="userLoading"
            @visible-change="handleUserDropdown"
          >
            <el-option
              v-for="u in users"
              :key="u.id"
              :label="`${u.name} (${u.username || u.email || u.id})`"
              :value="u.id"
            />
          </el-select>
        </el-form-item>
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

const isAdmin = ref(localStorage.getItem('role') === 'admin')
const groups = ref([])
const loading = ref(false)
const keyword = ref('')
const dialogVisible = ref(false)
const editingId = ref(0)
const selectedRows = ref([])
const users = ref([])
const userLoading = ref(false)
const selectedUserId = ref('')
const form = reactive({
  user_id: '',
  name: '',
  remark: ''
})

const dialogTitle = computed(() => (editingId.value ? '编辑分组' : '添加分组'))

const searchUsers = async (search = '') => {
  if (!isAdmin.value) return
  userLoading.value = true
  try {
    const res = await request.get('/users', { params: { keyword: search, pageSize: 200 } })
    users.value = (res.data?.list || res.list || []).map(item => ({
      ...item,
      username: item.username || item.email || item.name
    }))
  } finally {
    userLoading.value = false
  }
}

const handleUserDropdown = visible => {
  if (visible && isAdmin.value && !users.value.length) {
    searchUsers('')
  }
}

const handleUserChange = userId => {
  selectedUserId.value = userId || ''
  groups.value = []
  selectedRows.value = []
  fetchGroups()
}

const fetchGroups = () => {
  if (isAdmin.value && !selectedUserId.value) {
    groups.value = []
    return
  }
  loading.value = true
  request.get('/forward_groups', {
    params: {
      keyword: keyword.value,
      ...(isAdmin.value && selectedUserId.value ? { user_id: Number(selectedUserId.value) } : {})
    }
  }).then(res => {
    groups.value = res.data?.list || []
    loading.value = false
  }).catch(() => {
    loading.value = false
  })
}

const openCreate = () => {
  if (isAdmin.value && !selectedUserId.value) {
    ElMessage.warning('请先选择用户')
    return
  }
  editingId.value = 0
  form.user_id = isAdmin.value ? Number(selectedUserId.value) || '' : ''
  form.name = ''
  form.remark = ''
  dialogVisible.value = true
}

const openEdit = row => {
  editingId.value = row.id
  form.user_id = Number(row.user_id || row.uid || selectedUserId.value) || ''
  form.name = row.name || ''
  form.remark = row.remark || ''
  dialogVisible.value = true
}

const submitForm = () => {
  if (isAdmin.value && !form.user_id) {
    ElMessage.warning('请选择用户')
    return
  }
  const payload = {
    id: editingId.value,
    user_id: isAdmin.value ? Number(form.user_id) || 0 : undefined,
    name: form.name,
    remark: form.remark
  }
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

onMounted(() => {
  if (isAdmin.value) {
    searchUsers('')
  }
  fetchGroups()
})
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

