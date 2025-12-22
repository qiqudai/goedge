<template>
  <div class="app-container">
    <div class="filter-container" style="margin-bottom: 20px;">
      <el-button type="primary" :icon="Plus" @click="handleCreate">添加线路组</el-button>
    </div>

    <el-table :data="list" border style="width: 100%;">
      <el-table-column label="ID" prop="id" width="80" align="center" />
      <el-table-column label="分组名称" prop="name" />
      <el-table-column label="代号" prop="code" />
      <el-table-column label="创建时间">
        <template #default="{row}">{{ new Date(row.created_at).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="操作" align="center" width="150">
        <template #default="{row}">
          <el-button type="danger" size="small" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="500px">
      <el-form ref="dataForm" :model="temp" label-position="left" label-width="100px" style="margin-left:20px;">
        <el-form-item label="名称" prop="name">
          <el-input v-model="temp.name" placeholder="例如: 电信 VIP" />
        </el-form-item>
        <el-form-item label="代号" prop="code">
          <el-input v-model="temp.code" placeholder="例如: telecom_vip" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogFormVisible = false">取消</el-button>
        <el-button type="primary" @click="createData">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const dialogFormVisible = ref(false)
const dialogStatus = ref('create')
const textMap = { create: '创建线路组' }
const temp = reactive({ name: '', code: '' })

const getList = () => {
    request.get('/node_groups').then(res => {
        list.value = res.data.list
    })
}

const handleCreate = () => {
    temp.name = ''
    temp.code = ''
    dialogFormVisible.value = true
}

const createData = () => {
    request.post('/node_groups', temp).then(() => {
        dialogFormVisible.value = false
        ElMessage.success('创建成功')
        getList()
    })
}

const handleDelete = (row) => {
    ElMessageBox.confirm('确认删除该分组?', '警告', { type: 'warning' })
    .then(() => {
        request.delete(`/node_groups/${row.id}`).then(() => {
            ElMessage.success('删除成功')
            getList()
        })
    })
}

onMounted(() => getList())
</script>
