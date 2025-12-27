<template>
  <div class="app-container">
    <div class="filter-container">
      <el-input v-model="filters.keyword" placeholder="操作/内容/IP" style="width: 240px;" />
      <el-button type="primary" @click="handleFilter">查询</el-button>
    </div>

    <AppTable
      :data="list"
      border
      style="width: 100%;"
      v-model:current-page="filters.page"
      v-model:page-size="filters.pageSize"
      :page-sizes="[10, 20, 50]"
      layout="total, sizes, prev, pager, next, jumper"
      :total="total"
      @size-change="handleFilter"
      @current-change="handleFilter"
    >
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="action" label="操作" min-width="180" />
      <el-table-column prop="content" label="内容" min-width="220" show-overflow-tooltip />
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="created_at" label="时间" width="180" />
    </AppTable>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted} from 'vue'
import request from '@/utils/request'

const list = ref([])
const total = ref(0)

const filters = reactive({
  keyword: '',
  page: 1,
  pageSize: 20
})

const handleFilter = () => {
  request.get('/logs/operation', { params: filters }).then(res => {
    list.value = res.data?.list || []
    total.value = res.data?.total || 0
  })
}

onMounted(() => handleFilter())

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


