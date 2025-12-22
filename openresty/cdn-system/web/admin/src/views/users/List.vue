<template>
  <div class="app-container">
    <div class="filter-container" style="margin-bottom: 20px;">
      <el-input v-model="listQuery.username" placeholder="用户名" style="width: 200px;" class="filter-item" @keyup.enter="handleFilter" />
      <el-button class="filter-item" type="primary" :icon="Search" @click="handleFilter">
        搜索
      </el-button>
      <el-button class="filter-item" style="margin-left: 10px;" type="primary" :icon="Edit" @click="handleCreate">
        添加用户
      </el-button>
    </div>

    <el-table
      v-loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      style="width: 100%;">
      
      <el-table-column label="ID" prop="id" sortable="custom" align="center" width="80">
        <template #default="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column label="用户名" min-width="150px">
        <template #default="{row}">
          <span class="link-type" @click="handleUpdate(row)">{{ row.username }}</span>
          <el-tag v-if="row.role === 'admin'" type="danger" size="small" style="margin-left: 5px">管理员</el-tag>
        </template>
      </el-table-column>

      <el-table-column label="邮箱" min-width="150px" align="center">
        <template #default="{row}">
          <span>{{ row.email }}</span>
        </template>
      </el-table-column>
      
      <el-table-column label="手机号" width="120px" align="center">
        <template #default="{row}">
          <span>{{ row.phone || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column label="QQ" width="110px" align="center">
        <template #default="{row}">
          <span>{{ row.qq || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column label="余额" width="110px" align="center">
        <template #default="{row}">
          <span>{{ row.balance }}</span>
        </template>
      </el-table-column>

      <el-table-column label="状态" class-name="status-col" width="100">
        <template #default="{row}">
          <el-switch
            v-model="row.status"
            :active-value="1"
            :inactive-value="0"
            @change="handleStatusChange(row)"
          />
        </template>
      </el-table-column>

      <el-table-column label="备注" align="center" min-width="150">
        <template #default="{row}">
          <span>{{ row.remark }}</span>
        </template>
      </el-table-column>

      <el-table-column label="操作" align="center" width="230" class-name="small-padding fixed-width">
        <template #default="{row}">
          <el-button type="primary" size="small" @click="handleUpdate(row)">
            编辑
          </el-button>
          <el-button v-if="row.status!='deleted'" size="small" type="danger" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search, Edit } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref([])
const listLoading = ref(true)
const listQuery = reactive({
  page: 1,
  limit: 20,
  username: undefined,
  sort: '+id'
})

const getList = () => {
  listLoading.value = true
  request({
    url: '/users',
    method: 'get',
    params: listQuery
  }).then(response => {
    // Expected response structure: { code: 0, data: { list: [], total: x } }
    if (response.data) {
        list.value = response.data.list
    }
    listLoading.value = false
  }).catch(() => {
    listLoading.value = false
  })
}

const handleFilter = () => {
  listQuery.page = 1
  getList()
}

const handleCreate = () => {
  ElMessage.info('创建用户功能开发中')
}

const handleUpdate = (row) => {
    ElMessage.info('编辑用户: ' + row.username)
}

const handleStatusChange = (row) => {
    request({
        url: `/users/${row.id}/status`,
        method: 'put',
        data: { status: row.status }
    }).then(() => {
        ElMessage.success('状态更新成功')
    })
}

const handleDelete = (row) => {
    ElMessageBox.confirm('确认删除该用户吗?', '警告', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
    }).then(() => {
        request({
            url: `/users/${row.id}`,
            method: 'delete'
        }).then(() => {
             ElMessage.success('Deleted Successfully')
             const index = list.value.indexOf(row)
             list.value.splice(index, 1)
        })
    })
}

onMounted(() => {
  getList()
})
</script>
