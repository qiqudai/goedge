<template>
  <div class="app-container">
    <div class="filter-container">
      <el-input v-model="filters.keyword" placeholder="标题/内容" style="width: 240px;" />
      <el-button type="primary" @click="loadList">查询</el-button>
      <el-button type="success" @click="openCreate">新增公告</el-button>
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
      <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.is_show ? 'success' : 'info'">{{ row.is_show ? '\u663e\u793a' : '\u9690\u85cf' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" align="center">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </AppTable>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="560px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="标题">
          <el-input v-model="form.title" />
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="form.content" type="textarea" rows="6" />
        </el-form-item>
        <el-form-item label="标题">
          <el-switch v-model="form.is_show" />
        </el-form-item>
        <el-form-item label="标题">
          <el-switch v-model="form.is_red" />
        </el-form-item>
        <el-form-item label="标题">
          <el-switch v-model="form.is_bold" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="save">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessageBox, ElMessage } from 'element-plus'

const list = ref([])
const total = ref(0)

const filters = reactive({
  keyword: '',
  page: 1,
  pageSize: 20
})

const dialogVisible = ref(false)
const form = reactive({
  id: null,
  title: '',
  content: '',
  is_show: true,
  is_red: false,
  is_bold: false
})

const dialogTitle = computed(() => (form.id ? '\u7f16\u8f91\u516c\u544a' : '\u65b0\u589e\u516c\u544a'))

const loadList = () => {
  request.get('/announcements', { params: filters }).then(res => {
    list.value = res.data?.list || []
    total.value = res.data?.total || 0
  })
}

const openCreate = () => {
  form.id = null
  form.title = ''
  form.content = ''
  form.is_show = true
  form.is_red = false
  form.is_bold = false
  dialogVisible.value = true
}

const openEdit = row => {
  form.id = row.id
  form.title = row.title
  form.content = row.content
  form.is_show = row.is_show
  form.is_red = row.is_red
  form.is_bold = row.is_bold
  dialogVisible.value = true
}

const save = () => {
  const payload = {
    title: form.title,
    content: form.content,
    is_show: form.is_show,
    is_red: form.is_red,
    is_bold: form.is_bold
  }
  const req = form.id
    ? request.put(`/announcements/${form.id}`, payload)
    : request.post('/announcements', payload)

  req.then(() => {
    ElMessage.success('\u4fdd\u5b58\u6210\u529f')
    dialogVisible.value = false
    loadList()
  })
}

const remove = row => {
  ElMessageBox.confirm('\u786e\u5b9a\u5220\u9664\uff1f', '\u63d0\u793a', {
    type: 'warning'
  }).then(() => {
    request.delete(`/announcements/${row.id}`).then(() => {
      ElMessage.success('\u4fdd\u5b58\u6210\u529f')
      loadList()
    })
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

