<template>
  <div class="app-container">
    <div class="filter-container">
      <el-select v-model="filters.type" placeholder="关键词" clearable style="width: 160px;">
        <el-option label="刷新URL" value="refresh_url" />
        <el-option label="刷新目录" value="refresh_dir" />
        <el-option label="预热" value="preheat" />
      </el-select>
      <el-input v-model="filters.keyword" placeholder="关键词" style="width: 240px;" />
      <el-button type="primary" @click="loadList">查询</el-button>
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
      @size-change="loadList"
      @current-change="loadList"
    >
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="type" label="类型" width="120" />
      <el-table-column prop="state" label="类型" width="120" />
      <el-table-column prop="data" label="数据" min-width="220" show-overflow-tooltip />
      <el-table-column prop="create_at" label="创建时间" width="180" />
    </AppTable>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted} from 'vue'
import request from '@/utils/request'

const list = ref([])
const total = ref(0)

const filters = reactive({
  type: '',
  keyword: '',
  page: 1,
  pageSize: 20
})

const loadList = () => {
  request.get('/tasks', { params: filters }).then(res => {
    list.value = res.data?.list || res.list || []
    total.value = res.data?.total || res.total || 0
  })
}

onMounted(() => loadList())

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



