<template>
  <div class="region-panel">
    <div class="filter-container node-actions">
      <el-button type="primary" @click="handleEdit()">新增区域</el-button>
      <el-button :disabled="!selectedRows.length" @click="handleDeleteBatch">删除</el-button>
    </div>

    <AppTable
      v-loading="loading"
      :data="list"
      persist-key="node-region-table"
      border
      fit
      highlight-current-row
      style="width: 100%;"
      @selection-change="(rows) => selectedRows = rows"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column prop="id" label="ID" width="80" align="center" />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="remark" label="备注" min-width="160" />
      <el-table-column prop="l2_check_port" label="L2检测端口" width="140" align="center" />
      <el-table-column prop="sort_order" label="排序" width="100" align="center" />
      <el-table-column prop="create_at" label="添加时间" min-width="160" />
      <el-table-column label="操作" width="120" align="center">
        <template #default="{ row }">
          <div style="display: flex; justify-content: center; gap: 8px;">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </AppTable>

    <el-dialog v-model="visible" :title="form.id ? '编辑区域' : '新增区域'" width="520px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
        <el-form-item label="L2检测端口"><el-input v-model.number="form.l2_check_port" /></el-form-item>
        <el-form-item label="排序"><el-input v-model.number="form.sort_order" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const loading = ref(false)
const selectedRows = ref([])
const visible = ref(false)
const form = reactive({ id: 0, name: '', remark: '', l2_check_port: 80, sort_order: 100 })

const fetchData = async () => {
  loading.value = true
  const res = await request.get('/regions')
  list.value = res.data?.list || []
  loading.value = false
}

const handleEdit = (row) => {
  if (row) Object.assign(form, row)
  else Object.assign(form, { id: 0, name: '', remark: '', l2_check_port: 80, sort_order: 100 })
  visible.value = true
}

const handleSubmit = async () => {
  if (form.id) await request.put(`/regions/${form.id}`, form)
  else await request.post('/regions', form)
  ElMessage.success('保存成功')
  visible.value = false
  fetchData()
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除?', '提示').then(async () => {
    await request.delete(`/regions/${row.id}`)
    fetchData()
  })
}

const handleDeleteBatch = () => {
    const ids = selectedRows.value.map(r => r.id)
    ElMessageBox.confirm(`确定删除选中的 ${ids.length} 个区域吗?`, '提示').then(async () => {
        await Promise.all(ids.map(id => request.delete(`/regions/${id}`)))
        fetchData()
    })
}

onMounted(fetchData)
</script>
