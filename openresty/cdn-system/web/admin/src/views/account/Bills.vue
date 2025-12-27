<template>
  <div class="app-container">
    <div class="filter-container">
      <el-select v-model="filters.type" placeholder="类型" style="width: 140px;">
        <el-option label="全部" value="" />
        <el-option label="购买" value="purchase" />
        <el-option label="续购" value="renew" />
        <el-option label="充值" value="recharge" />
      </el-select>
      <el-input v-model="filters.keyword" placeholder="订单号/备注" style="width: 240px;" />
      <el-button type="primary" @click="applyFilter">查询</el-button>
    </div>

    <AppTable
      :data="list"
      border
      style="width: 100%;"
      v-model:current-page="filters.page"
      v-model:page-size="filters.pageSize"
      :page-sizes="[10, 20, 30]"
      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="applyFilter"
      @current-change="applyFilter"
    >
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="type_label" label="类型" width="120" />
      <el-table-column prop="remark" label="备注" min-width="200" />
      <el-table-column prop="price" label="类型" width="120" />
      <el-table-column prop="pay" label="实际支付" width="120" />
      <el-table-column prop="more" label="更多" min-width="180" />
      <el-table-column prop="pay_type" label="支付方式" width="140" />
      <el-table-column prop="order_no" label="订单号" min-width="200" />
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column label="已付款" width="100" align="center">
        <template #default="{ row }">
          <el-tag :type="row.paid ? 'success' : 'info'">{{ row.paid ? '\u5df2\u4ed8\u6b3e' : '\u672a\u4ed8\u6b3e' }}</el-tag>
        </template>
      </el-table-column>
    </AppTable>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'

const list = ref([])
const total = ref(0)

const filters = reactive({
  type: '',
  keyword: '',
  page: 1,
  pageSize: 10
})

const applyFilter = () => {
  request.get('/orders', { params: filters }).then(res => {
    list.value = res.data?.list || []
    total.value = res.data?.total || 0
  })
}

onMounted(() => applyFilter())
</script>

<style scoped>
.filter-container {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  align-items: center;
  margin-bottom: 16px;
}
.pagination-container {
  margin-top: 16px;
  text-align: right;
}
</style>

