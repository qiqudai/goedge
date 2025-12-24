<template>
  <div class="app-container">
    <div class="filter-container" style="margin-bottom: 20px;">
      <el-input v-model="listQuery.keyword" :placeholder="t.keywordPlaceholder" style="width: 200px;" class="filter-item" @keyup.enter="handleFilter" />
      <el-button class="filter-item" type="primary" :icon="Search" @click="handleFilter">
        {{ t.search }}
      </el-button>
      <el-button class="filter-item" style="margin-left: 10px;" type="primary" :icon="Edit" @click="handleCreate">
        {{ t.addUser }}
      </el-button>
    </div>

    <el-table
      v-loading="listLoading"
      :data="list"
      border
      fit
      highlight-current-row
      style="width: 100%;"
    >
      <el-table-column label="ID" prop="id" sortable="custom" align="center" width="80">
        <template #default="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="t.userName" min-width="150">
        <template #default="{ row }">
          <span class="link-type" @click="handleUpdate(row)">{{ row.name }}</span>
          <el-tag v-if="row.type === 1" type="danger" size="small" style="margin-left: 5px">{{ t.admin }}</el-tag>
        </template>
      </el-table-column>

      <el-table-column :label="t.email" min-width="150" align="center">
        <template #default="{ row }">
          <span>{{ row.email }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="t.phone" width="120" align="center">
        <template #default="{ row }">
          <span>{{ row.phone || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column label="QQ" width="110" align="center">
        <template #default="{ row }">
          <span>{{ row.qq || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="t.balance" width="110" align="center">
        <template #default="{ row }">
          <span>{{ row.balance }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="t.status" class-name="status-col" width="100">
        <template #default="{ row }">
          <el-switch
            v-model="row.status"
            :active-value="1"
            :inactive-value="0"
            @change="handleStatusChange(row)"
          />
        </template>
      </el-table-column>

      <el-table-column :label="t.remark" align="center" min-width="150">
        <template #default="{ row }">
          <span>{{ row.des || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="t.action" align="center" width="230" class-name="small-padding fixed-width">
        <template #default="{ row }">
          <el-button type="primary" size="small" @click="handleUpdate(row)">
            {{ t.edit }}
          </el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">
            {{ t.delete }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick } from 'vue'
import { Search, Edit } from '@element-plus/icons-vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

const t = {
  keywordPlaceholder: '\u7528\u6237\u540d/\u90ae\u7bb1/\u624b\u673a\u53f7',
  search: '\u641c\u7d22',
  addUser: '\u6dfb\u52a0\u7528\u6237',
  userName: '\u7528\u6237\u540d',
  admin: '\u7ba1\u7406\u5458',
  email: '\u90ae\u7bb1',
  phone: '\u624b\u673a\u53f7',
  balance: '\u4f59\u989d',
  status: '\u72b6\u6001',
  remark: '\u5907\u6ce8',
  action: '\u64cd\u4f5c',
  edit: '\u7f16\u8f91',
  delete: '\u5220\u9664',
  createUserTip: '\u521b\u5efa\u7528\u6237\u529f\u80fd\u5f00\u53d1\u4e2d',
  editUserTip: '\u7f16\u8f91\u7528\u6237: ',
  statusUpdated: '\u72b6\u6001\u66f4\u65b0\u6210\u529f',
  deleteConfirm: '\u786e\u8ba4\u5220\u9664\u8be5\u7528\u6237?',
  warning: '\u8b66\u544a',
  confirm: '\u786e\u5b9a',
  cancel: '\u53d6\u6d88',
  deleteSuccess: '\u5220\u9664\u6210\u529f'
}

const list = ref([])
const listLoading = ref(true)
const initSwitchLock = ref(true)
const listQuery = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  sort: '+id'
})

const getList = () => {
  listLoading.value = true
  initSwitchLock.value = true
  request({
    url: '/users',
    method: 'get',
    params: listQuery
  }).then(response => {
    const data = response.data || {}
    const items = data.list || []
    list.value = items.map(item => ({
      ...item,
      status: item.enable ? 1 : 0,
      name: item.name || item.username || ''
    }))
  }).finally(() => {
    listLoading.value = false
    nextTick(() => {
      initSwitchLock.value = false
    })
  })
}

const handleFilter = () => {
  listQuery.page = 1
  getList()
}

const handleCreate = () => {
  ElMessage.info(t.createUserTip)
}

const handleUpdate = row => {
  ElMessage.info(t.editUserTip + (row.name || row.username || row.id))
}

const handleStatusChange = row => {
  if (initSwitchLock.value) {
    return
  }
  request({
    url: `/users/${row.id}/status`,
    method: 'put',
    data: { status: row.status }
  }).then(() => {
    ElMessage.success(t.statusUpdated)
  })
}

const handleDelete = row => {
  ElMessageBox.confirm(t.deleteConfirm, t.warning, {
    confirmButtonText: t.confirm,
    cancelButtonText: t.cancel,
    type: 'warning'
  }).then(() => {
    request({
      url: `/users/${row.id}`,
      method: 'delete'
    }).then(() => {
      ElMessage.success(t.deleteSuccess)
      const index = list.value.indexOf(row)
      list.value.splice(index, 1)
    })
  })
}

onMounted(() => {
  getList()
})
</script>
