<template>
  <div class="app-container">
    <div class="filter-container">
      <el-input v-model="filters.keyword" placeholder="??? / ??ID" style="width: 220px;" />
      <el-button type="primary" @click="handleFilter">??</el-button>
      <el-button type="success" @click="dialogVisible = true">????</el-button>
    </div>

    <el-table :data="list" border style="width: 100%; margin-top: 20px;">
      <el-table-column prop="id" label="???" width="120" />
      <el-table-column prop="user_id" label="??ID" width="100" />
      <el-table-column prop="amount" label="??" width="140">
        <template #default="{ row }">?{{ row.amount.toFixed(2) }}</template>
      </el-table-column>
      <el-table-column prop="status" label="??" width="120">
        <template #default="{ row }">
          <el-tag type="success" v-if="row.status === 1">???</el-tag>
          <el-tag type="info" v-else>???</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="pay_type" label="????" width="120" />
      <el-table-column prop="order_no" label="?????" min-width="180" />
      <el-table-column prop="type" label="??" width="120" />
      <el-table-column prop="remark" label="??" min-width="200" show-overflow-tooltip />
      <el-table-column prop="created_at" label="??" width="180" />
    </el-table>

    <div class="pagination-container">
      <el-pagination
        v-model:current-page="filters.page"
        v-model:page-size="filters.pageSize"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleFilter"
        @current-change="handleFilter"
      />
    </div>

    <el-dialog title="????" v-model="dialogVisible" width="420px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="??ID">
          <el-input v-model.number="form.user_id" />
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model.number="form.amount" type="number" />
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model="form.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">??</el-button>
        <el-button type="primary" @click="handleRecharge">??</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const list = ref([])
const total = ref(0)
const dialogVisible = ref(false)
const form = reactive({ user_id: undefined, amount: 0, remark: '' })

const filters = reactive({
  keyword: '',
  page: 1,
  pageSize: 20
})

const getList = () => {
  request.get('/orders', { params: filters }).then(res => {
    list.value = res.data?.list || []
    total.value = res.data?.total || 0
  })
}

const handleFilter = () => {
  getList()
}

const handleRecharge = () => {
  if (!form.user_id || !form.amount) {
    ElMessage.warning('?????ID???')
    return
  }
  request.post('/recharge', form).then(() => {
    ElMessage.success('????')
    dialogVisible.value = false
    getList()
  })
}

onMounted(() => getList())
</script>

<style scoped>
.filter-container {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
}
.pagination-container {
  margin-top: 16px;
  text-align: right;
}
</style>
