<template>
  <div class="app-container">
    <div class="filter-container">
      <el-button type="primary" @click="openCreate">添加分组</el-button>
    </div>

    <el-table v-loading="loading" :data="groups" border style="width: 100%;">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="分组名称" min-width="200" />
      <el-table-column prop="remark" label="备注" min-width="200" />
      <el-table-column label="操作" width="160" align="center">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" size="small" @click="removeGroup(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="420px">
      <el-form :model="form" label-width="80px">
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
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const groups = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const editingId = ref(0)
const form = reactive({
  name: '',
  remark: ''
})

const dialogTitle = computed(() => (editingId.value ? '编辑分组' : '添加分组'))

const fetchGroups = () => {
  loading.value = true
  request.get('/site_groups').then(res => {
    groups.value = res.data?.list || res.list || []
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
  const payload = { name: form.name, remark: form.remark }
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
  ElMessageBox.confirm('确认删除该分组?', '提示', {
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

onMounted(fetchGroups)
</script>
