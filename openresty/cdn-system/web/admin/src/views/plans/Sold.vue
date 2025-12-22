<template>
  <div class="app-container">
    <h2>已售套餐</h2>

    <el-table :data="list" v-loading="loading" border style="width: 100%; margin-top: 20px;">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="user_id" label="用户ID" width="100" />
      <el-table-column prop="plan_name" label="套餐名称" />
      <el-table-column prop="expire_at" label="到期时间" />
      <el-table-column prop="status" label="状态" width="100">
         <template #default="{row}">
             <el-tag :type="row.status === 'active' ? 'success' : 'danger'">{{ row.status }}</el-tag>
         </template>
      </el-table-column>
      <el-table-column prop="created_at" label="购买时间" />
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '@/utils/request'

const list = ref([])
const loading = ref(false)

const fetchList = () => {
    loading.value = true
    request.get('/user_plans').then(res => {
        list.value = res.data.list || []
    }).finally(() => {
        loading.value = false
    })
}

onMounted(() => {
    fetchList()
})
</script>
