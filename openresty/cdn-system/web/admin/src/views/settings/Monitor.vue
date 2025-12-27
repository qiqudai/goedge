<template>
  <div class="app-container">
    <el-form label-width="160px" :model="form">
      <el-form-item label="通知时间段">
        <el-input v-model="form.notification_period" placeholder="8-22" />
      </el-form-item>
      <el-form-item label="通知方式">
        <el-select v-model="form.notify_method" placeholder="请选择">
          <el-option label="邮件" value="email" />
          <el-option label="短信" value="sms" />
          <el-option label="邮件+短信" value="email sms" />
        </el-select>
      </el-form-item>
      <el-form-item label="通知类型">
        <el-checkbox-group v-model="notifyMsgTypes">
          <el-checkbox value="node_ip_dns">节点IP变化</el-checkbox>
          <el-checkbox value="bandwidth">带宽超限</el-checkbox>
          <el-checkbox value="backup_ip">备用IP</el-checkbox>
          <el-checkbox value="backup_default_line">默认线路备份</el-checkbox>
          <el-checkbox value="backup_group">节点组备份</el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item label="邮箱">
        <el-input v-model="form.email" placeholder="admin@example.com" />
      </el-form-item>
      <el-form-item label="手机号">
        <el-input v-model="form.phone" placeholder="+86138..." />
      </el-form-item>
      <el-form-item label="带宽超限次数">
        <el-input-number v-model="form.bw_exceed_times" :min="1" />
      </el-form-item>
      <el-form-item label="监控API">
        <el-input v-model="form.monitor_api" placeholder="http://..." />
      </el-form-item>
      <el-form-item label="检测间隔(秒)">
        <el-input-number v-model="form.interval" :min="10" />
      </el-form-item>
      <el-form-item label="失败次数">
        <el-input-number v-model="form.failed_times" :min="1" />
      </el-form-item>
      <el-form-item label="失败率(%)">
        <el-input v-model="form.failed_rate" placeholder="50" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="saveConfig">保存配置</el-button>
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
    ElMessage.success('\u4fdd\u5b58\u6210\u529f')
  })
}

onMounted(() => getConfig())
</script>
