<template>
  <div class="app-container">
    <el-form label-width="160px" :model="form">
      <el-form-item label="?????">
        <el-input v-model="form.notification_period" placeholder="8-22" />
      </el-form-item>
      <el-form-item label="????">
        <el-select v-model="form.notify_method" placeholder="???">
          <el-option label="??" value="email" />
          <el-option label="??" value="sms" />
          <el-option label="??+??" value="email sms" />
        </el-select>
      </el-form-item>
      <el-form-item label="????">
        <el-checkbox-group v-model="notifyMsgTypes">
          <el-checkbox label="node_ip_dns">??IP??</el-checkbox>
          <el-checkbox label="bandwidth">????</el-checkbox>
          <el-checkbox label="backup_ip">??IP</el-checkbox>
          <el-checkbox label="backup_default_line">??????</el-checkbox>
          <el-checkbox label="backup_group">?????</el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item label="??">
        <el-input v-model="form.email" placeholder="admin@example.com" />
      </el-form-item>
      <el-form-item label="?????">
        <el-input v-model="form.phone" placeholder="+86138..." />
      </el-form-item>
      <el-form-item label="??????">
        <el-input-number v-model="form.bw_exceed_times" :min="1" />
      </el-form-item>
      <el-form-item label="??API">
        <el-input v-model="form.monitor_api" placeholder="http://..." />
      </el-form-item>
      <el-form-item label="????(?)">
        <el-input-number v-model="form.interval" :min="10" />
      </el-form-item>
      <el-form-item label="??????">
        <el-input-number v-model="form.failed_times" :min="1" />
      </el-form-item>
      <el-form-item label="?????(%)">
        <el-input v-model="form.failed_rate" placeholder="50" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="saveConfig">????</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { reactive, onMounted, ref, watch } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const form = reactive({
  notification_period: '8-22',
  notify_method: 'email sms',
  notify_msg_type: '',
  email: '',
  phone: '',
  bw_exceed_times: 2,
  monitor_api: '',
  interval: 30,
  failed_times: 3,
  failed_rate: '50'
})

const notifyMsgTypes = ref([])

const syncMsgTypes = () => {
  form.notify_msg_type = notifyMsgTypes.value.join(' ')
}

watch(notifyMsgTypes, () => syncMsgTypes(), { deep: true })

const getConfig = () => {
  request.get('/monitor_config').then(res => {
    if (res.data) {
      Object.assign(form, res.data)
      if (form.notify_msg_type) {
        notifyMsgTypes.value = String(form.notify_msg_type).split(' ').filter(Boolean)
      } else {
        notifyMsgTypes.value = []
      }
    }
  })
}

const saveConfig = () => {
  syncMsgTypes()
  request.post('/monitor_config', form).then(() => {
    ElMessage.success('????')
  })
}

onMounted(() => getConfig())
</script>
