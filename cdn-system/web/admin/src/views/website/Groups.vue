<template>
  <div class="app-container">
    <el-card>
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
            v-model="listQuery.keyword"
            placeholder="输入分组名称搜索"
            style="width: 200px;"
            class="filter-item"
            @keyup.enter="handleFilter"
          />
          <el-button type="primary" class="filter-item" @click="handleFilter">搜索</el-button>
        </div>
      </div>

      <AppTable
        :data="groups"
        :loading="loading"
        border
        persist-key="website-groups"
        storage-key="website-groups"
        style="width: 100%;"
        v-model:current-page="listQuery.page"
        v-model:page-size="listQuery.pageSize"

        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleFilter"
        @current-change="handleFilter"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="50" align="center" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="分组名称" min-width="200" />
        <el-table-column prop="remark" label="备注" min-width="200" />
        <el-table-column label="操作" width="160" align="center">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="removeGroup(row)">删除</el-button>
          </template>
        </el-table-column>
      </AppTable>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="420px">
      <el-form :model="form" label-width="80px">
        <el-form-item v-if="isAdmin" label="用户">
          <el-select
            v-model="form.uid"
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
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" />
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
import { ref, reactive, onMounted, computed} from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const isAdmin = ref(localStorage.getItem('role') === 'admin')
const groups = ref([])
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const editingId = ref(0)
const selectedRows = ref([])
const users = ref([])
const userLoading = ref(false)
const selectedUserId = ref('')
const form = reactive({
  uid: '',
  name: '',
  remark: ''
})

const listQuery = reactive({
  page: 1,
  pageSize: 10,
  keyword: ''
})

const dialogTitle = computed(() => (editingId.value ? '编辑分组' : '添加分组'))

const searchUsers = async (keyword = '') => {
  if (!isAdmin.value) return
  userLoading.value = true
  try {
    const res = await request.get('/users', { params: { keyword, pageSize: 200 } })
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
  listQuery.page = 1
  fetchGroups()
}

const fetchGroups = () => {
  if (isAdmin.value && !selectedUserId.value) {
    groups.value = []
    total.value = 0
    return
  }
  loading.value = true
  request.get('/site_groups', {
    params: {
      page: listQuery.page,
      pageSize: listQuery.pageSize,
      keyword: listQuery.keyword,
      ...(isAdmin.value && selectedUserId.value ? { user_id: Number(selectedUserId.value) } : {})
    }
  }).then(res => {
    groups.value = res.data?.list || res.list || []
    total.value = res.data?.total || res.total || 0
    loading.value = false
  }).catch(() => {
    loading.value = false
  })
}

const handleFilter = () => {
  listQuery.page = 1
  fetchGroups()
}


const openCreate = () => {
  if (isAdmin.value && !selectedUserId.value) {
    ElMessage.warning('请先选择用户')
    return
  }
  editingId.value = 0
  form.uid = isAdmin.value ? Number(selectedUserId.value) || '' : ''
  form.name = ''
  form.remark = ''
  dialogVisible.value = true
}

const openEdit = row => {
  editingId.value = row.id
  form.uid = Number(row.uid || selectedUserId.value) || ''
  form.name = row.name || ''
  form.remark = row.remark || ''
  dialogVisible.value = true
}

const submitForm = () => {
  if (isAdmin.value && !form.uid) {
    ElMessage.warning('请选择用户')
    return
  }
  const payload = {
    uid: isAdmin.value ? Number(form.uid) || 0 : undefined,
    name: form.name,
    remark: form.remark
  }
  if (editingId.value) {
    request.put(`/site_groups/${editingId.value}`, payload).then(() => {
      ElMessage.success('更新成功')
      dialogVisible.value = false
      fetchGroups()
    })
  } else {
    request.post('/site_groups', payload).then(() => {
      ElMessage.success('创建成功')
      dialogVisible.value = false
      fetchGroups()
    })
  }
}

const removeGroup = row => {
  ElMessageBox.confirm('确认删除该分组?', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.delete(`/site_groups/${row.id}`).then(() => {
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
      const promises = selectedRows.value.map(row => request.delete(`/site_groups/${row.id}`))
      await Promise.all(promises)
      ElMessage.success('批量删除成功')
      fetchGroups()
      selectedRows.value = []
    } catch (err) {
      //
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
.filter-container {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 20px;
}
.filter-left, .filter-right {
  display: flex;
  gap: 10px;
}
.pagination-container {
  margin-top: 20px;
  text-align: right;
  display: flex;
  justify-content: flex-end;
}
</style>


