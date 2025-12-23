<template>
  <div class="app-container">
    <el-table :data="list" border style="width: 100%;">
      <el-table-column prop="name" label="消息类型" min-width="220" />
      <el-table-column label="手机提醒" width="160" align="center">
        <template #default="{ row }">
          <el-checkbox v-model="row.phone" />
        </template>
      </el-table-column>
      <el-table-column label="邮件提醒" width="160" align="center">
        <template #default="{ row }">
          <el-checkbox v-model="row.email" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" align="center">
        <template #default>
          <el-button type="primary" link @click="save">保存</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const list = ref([])

const load = () => {
  request.get('/message_sub').then(res => {
    list.value = res.data?.list || []
  })
}

const save = () => {
  request.put('/message_sub', { list: list.value.map(item => ({
    msg_type: item.msg_type,
    phone: item.phone,
    email: item.email
  })) }).then(() => {
    ElMessage.success('\u4fdd\u5b58\u6210\u529f')
  })
}

onMounted(() => load())
</script>
