<template>
  <div class="app-container">
    <el-table :data="list" border style="width: 100%;">
      <el-table-column prop="name" label="消息类型" min-width="220" />
      <el-table-column label="手机提醒" width="160" align="center">
        <template #default="{ row }">
          <el-checkbox v-model="row.phone" @change="saveAndReload" />
        </template>
      </el-table-column>
      <el-table-column label="邮件提醒" width="160" align="center">
        <template #default="{ row }">
          <el-checkbox v-model="row.email" @change="saveAndReload" />
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
const saving = ref(false)

const load = () => {
  request.get('/message_sub').then(res => {
    list.value = res.data?.list || []
  })
}

const saveAndReload = () => {
  if (saving.value) return
  saving.value = true
  request.put('/message_sub', { list: list.value.map(item => ({
    msg_type: item.msg_type,
    phone: item.phone,
    email: item.email
  })) }).then(() => {
    ElMessage.success('保存成功')
    load()
  }).finally(() => {
    saving.value = false
  })
}

onMounted(() => load())
</script>
