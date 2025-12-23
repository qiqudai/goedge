<template>
  <div class="app-container">
    <div class="filter-container">
      <el-select v-model="filters.type" placeholder="消息类型" style="width: 180px;">
        <el-option label="全部" value="" />
        <el-option label="套餐到期" value="package-expire" />
        <el-option label="流量超限" value="traffic-exceed" />
        <el-option label="连接数超限" value="connection-exceed" />
        <el-option label="带宽超限" value="bandwidth-exceed" />
        <el-option label="防护规则切换" value="cc-switch" />
        <el-option label="证书到期" value="cert-expire" />
      </el-select>
      <el-input v-model="filters.keyword" placeholder="标题/用户/网站ID" style="width: 260px;" />
      <el-button type="primary" @click="loadList">查询</el-button>
    </div>

    <el-table :data="list" border style="width: 100%;">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="type_label" label="类型" width="160" />
      <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
      <el-table-column prop="site_id" label="网站ID" width="120" />
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column label="操作" width="120" align="center">
        <template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-container">
      <el-pagination
        v-model:current-page="filters.page"
        v-model:page-size="filters.pageSize"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="loadList"
        @current-change="loadList"
      />
    </div>

    <el-dialog v-model="detailVisible" title="消息详情" width="560px">
      <el-form label-width="80px">
        <el-form-item label="标题">
          <div>{{ detail.title }}</div>
        </el-form-item>
        <el-form-item label="邮件内容">
          <div class="detail-content" v-html="detail.content"></div>
        </el-form-item>
        <el-form-item label="邮件内容">
          <div class="detail-content" v-html="detail.phone"></div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
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
  pageSize: 20
})

const detailVisible = ref(false)
const detail = reactive({
  title: '',
  content: '',
  phone: ''
})

const loadList = () => {
  request.get('/messages', { params: filters }).then(res => {
    const rows = res.data?.list || []
    list.value = rows.map(item => ({
      ...item,
      type_label: item.type_label || item.type
    }))
    total.value = res.data?.total || 0
  })
}

const openDetail = row => {
  detail.title = row.title
  detail.content = row.content
  detail.phone = row.phone
  detailVisible.value = true
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
.detail-content {
  padding: 8px 10px;
  background: #f5f7fa;
  border-radius: 4px;
  line-height: 1.6;
}
</style>
