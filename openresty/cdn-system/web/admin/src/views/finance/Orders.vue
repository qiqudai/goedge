<template>
  <div class="app-container">
    <div class="filter-container">
      <el-input v-model="filters.keyword" placeholder="用户名/用户ID" style="width: 220px;" />
      <el-button type="primary" @click="handleFilter">查询</el-button>
      <el-button type="success" @click="dialogVisible = true">手动充值</el-button>
    </div>

    <el-table :data="list" border style="width: 100%; margin-top: 20px;">
      <el-table-column prop="id" label="订单ID" width="120" />
      <el-table-column prop="user_id" label="用户ID" width="100" />
      <el-table-column prop="amount" label="金额" width="140">
        <template #default="{ row }">¥{{ row.amount.toFixed(2) }}</template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag type="success" v-if="row.status === 1">已支付</el-tag>
          <el-tag type="info" v-else>已支付</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="pay_type" label="支付方式" width="120" />
      <el-table-column prop="order_no" label="订单号" min-width="180" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column prop="remark" label="备注" min-width="200" show-overflow-tooltip />
      <el-table-column prop="created_at" label="创建时间" width="180" />
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

    <el-dialog title="手动充值" v-model="dialogVisible" width="420px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="用户ID">
          <el-input v-model.number="form.user_id" />
        </el-form-item>
        <el-form-item label="金额">
          <el-input v-model.number="form.amount" type="number" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleRecharge">确定</el-button>
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
    ElMessage.warning('\u8bf7\u8f93\u5165\u7528\u6237ID\u548c\u91d1\u989d')
    return
  }
  request.post('/recharge', form).then(() => {
    ElMessage.success('\u5145\u503c\u6210\u529f')
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
