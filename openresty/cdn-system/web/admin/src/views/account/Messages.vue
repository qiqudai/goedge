<template>
  <div class="app-container">
    <div class="filter-container">
      <el-select v-model="filters.type" placeholder="消息类型" style="width: 140px;">
        <el-option label="全部" value="" />
        <el-option label="带宽超限" value="bandwidth" />
        <el-option label="流量超限" value="traffic" />
        <el-option label="套餐到期" value="package" />
      </el-select>
      <el-input v-model="filters.keyword" placeholder="标题/网站ID" style="width: 240px;" />
      <el-button type="primary" @click="applyFilter">查询</el-button>
    </div>

    <el-table :data="list" border style="width: 100%;">
      <el-table-column prop="user_id" label="所属用户ID" width="120" />
      <el-table-column prop="type_label" label="类型" width="140" />
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
        :page-sizes="[10, 20, 30]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="applyFilter"
        @current-change="applyFilter"
      />
    </div>

    <el-dialog v-model="detailVisible" title="消息详情" width="560px">
      <el-form label-width="80px">
        <el-form-item label="标题">
          <div>{{ detail.title }}</div>
        </el-form-item>
        <el-form-item label="邮件内容">
          <div class="detail-content" v-html="detail.email"></div>
        </el-form-item>
        <el-form-item label="短信内容">
          <div class="detail-content" v-html="detail.sms"></div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="detailVisible = false">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'

const allRows = ref([
  {
    user_id: 17,
    type: 'bandwidth',
    type_label: '带宽超限',
    title: 'cdn套餐带宽超限提醒!',
    site_id: 14,
    created_at: '2025-12-22 10:45:33',
    email: '<p>尊敬的feiyang666：</p><p>您的套餐( ID: 14，名称商业版K(畅) )当前带宽为249.39Mbps，已超过限制的50.0Mbps，现系统已开启限速。</p>',
    sms: '【cdn】尊敬的feiyang666，您的套餐( ID: 14，名称商业版K(畅) )当前带宽为249.39Mbps，已超过限制的50.0Mbps，现系统已开启限速。'
  },
  {
    user_id: 17,
    type: 'traffic',
    type_label: '流量超限',
    title: 'cdn套餐流量超限提醒!',
    site_id: 14,
    created_at: '2025-12-11 14:22:00',
    email: '<p>尊敬的feiyang666：</p><p>您的套餐流量已接近上限，请及时充值。</p>',
    sms: '【cdn】您的套餐流量已接近上限，请及时充值。'
  }
])

const list = ref([...allRows.value])
const total = ref(allRows.value.length)

const filters = reactive({
  type: '',
  keyword: '',
  page: 1,
  pageSize: 10
})

const detailVisible = ref(false)
const detail = reactive({
  title: '',
  email: '',
  sms: ''
})

const applyFilter = () => {
  let filtered = allRows.value
  if (filters.type) {
    filtered = filtered.filter(row => row.type === filters.type)
  }
  if (filters.keyword) {
    const keyword = filters.keyword.trim()
    filtered = filtered.filter(row => row.title.includes(keyword) || String(row.site_id).includes(keyword))
  }
  total.value = filtered.length
  const start = (filters.page - 1) * filters.pageSize
  list.value = filtered.slice(start, start + filters.pageSize)
}

const openDetail = row => {
  detail.title = row.title
  detail.email = row.email
  detail.sms = row.sms
  detailVisible.value = true
}
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
