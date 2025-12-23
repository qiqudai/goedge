<template>
  <div class="app-container">
    <div class="filter-container">
      <el-select v-model="filters.type" placeholder="????" style="width: 140px;">
        <el-option label="??" value="" />
        <el-option label="??" value="purchase" />
        <el-option label="??" value="renew" />
        <el-option label="??" value="recharge" />
      </el-select>
      <el-input v-model="filters.keyword" placeholder="???/??" style="width: 240px;" />
      <el-button type="primary" @click="applyFilter">??</el-button>
    </div>

    <el-table :data="list" border style="width: 100%;">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="type_label" label="??" width="120" />
      <el-table-column prop="remark" label="??" min-width="200" />
      <el-table-column prop="price" label="??" width="120" />
      <el-table-column prop="pay" label="????" width="120" />
      <el-table-column prop="more" label="??" min-width="180" />
      <el-table-column prop="pay_type" label="????" width="140" />
      <el-table-column prop="order_no" label="???" min-width="200" />
      <el-table-column prop="created_at" label="????" width="180" />
      <el-table-column label="???" width="100" align="center">
        <template #default="{ row }">
          <el-tag :type="row.paid ? 'success' : 'info'">{{ row.paid ? '???' : '???' }}</el-tag>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-container">
      <el-pagination
        v-model:current-page="filters.page"
        v-model:page-size="filters.pageSize"
        :page-sizes="[10, 20, 30]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="applyFilter"
        @current-change="applyFilter"
      />
    </div>
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
