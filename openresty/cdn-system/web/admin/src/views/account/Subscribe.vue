<template>
  <div class="app-container">
    <el-table :data="list" border style="width: 100%;">
      <el-table-column prop="name" label="消息类型" min-width="220" />
      <el-table-column label="手机提醒" width="160" align="center">
        <template #default="{ row }">
          <el-checkbox v-model="row.sms" />
        </template>
      </el-table-column>
      <el-table-column label="邮件提醒" width="160" align="center">
        <template #default="{ row }">
          <el-checkbox v-model="row.email" />
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const list = ref([
  { name: '套餐到期', sms: true, email: true },
  { name: '流量超限', sms: true, email: true },
  { name: '连接数超限', sms: true, email: true },
  { name: '带宽超限', sms: true, email: true },
  { name: '防护规则切换', sms: true, email: true },
  { name: '证书到期', sms: true, email: true }
])
</script>
