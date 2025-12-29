<template>
  <div class="filter-container">
    <el-button type="primary" size="small" @click="handleCreate">新增ACL</el-button>
    <el-input v-model="query.name" placeholder="名称" style="width: 150px; margin-left: 10px;" size="small" />
    <el-button size="small" type="primary" :icon="Search" @click="fetchData">查询</el-button>
  </div>

  <AppTable :data="list" border fit highlight-current-row persist-key="acl-rules">
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="name" label="名称" />
    <el-table-column label="开启" width="80" align="center">
      <template #default="{row}">
        <span :class="row.is_on ? 'text-success' : 'text-danger'">{{ row.is_on ? '启用' : '禁用' }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="create_time" label="创建时间" width="160" />
    <el-table-column label="操作" width="120" align="center">
      <template #default="{row}">
        <el-button type="primary" link size="small" @click="handleEdit(row)">编辑</el-button>
        <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
      </template>
    </el-table-column>
  </AppTable>

  <el-dialog :title="dialogMode === 'create' ? '新增ACL' : '编辑ACL'" v-model="dialogVisible" width="600px">
     <el-form :model="form" label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="form.is_on" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>
     </el-form>
     <template #footer>
        <el-button size="small" @click="dialogVisible = false">取消</el-button>
        <el-button size="small" type="primary" @click="submitForm">确定</el-button>
     </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const query = reactive({ name: '' })
const dialogVisible = ref(false)
const dialogMode = ref('create')
const form = reactive({ id: 0, name: '', is_on: true, remark: '' })

const fetchData = async () => {
  const { data } = await request.get('/rules/acl', { params: query })
  list.value = data.list || []
}

const handleCreate = () => {
  dialogMode.value = 'create'
  Object.assign(form, { id: 0, name: '', is_on: true, remark: '' })
  dialogVisible.value = true
}

const handleEdit = (row) => {
  dialogMode.value = 'update'
  Object.assign(form, row)
  dialogVisible.value = true
}

const submitForm = async () => {
    if (form.id) await request.put(`/rules/acl/${form.id}`, form)
    else await request.post('/rules/acl', form)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除?', '提示').then(async () => {
    await request.delete(`/rules/acl/${row.id}`)
    fetchData()
  })
}

onMounted(fetchData)
</script>
