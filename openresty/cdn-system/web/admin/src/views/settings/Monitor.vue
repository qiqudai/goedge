<template>
  <div class="app-container">
    <el-tabs type="border-card">
      <el-tab-pane label="通知渠道">
        <el-form label-width="150px">
          <el-form-item label="邮件地址">
            <el-input v-model="form.notify_email" placeholder="alert@example.com" />
          </el-form-item>
          <el-form-item label="Telegram 群组 ID">
            <el-input v-model="form.notify_telegram" placeholder="-100xxxxxxx" />
          </el-form-item>
          <el-form-item label="短信手机号">
            <el-input v-model="form.notify_phone" placeholder="+86138..." />
          </el-form-item>
        </el-form>
      </el-tab-pane>
      
      <el-tab-pane label="消息模板">
        <el-collapse>
          <el-collapse-item title="IP 恢复模板" name="1">
            <el-input type="textarea" :rows="3" v-model="form.template_ip_up" />
          </el-collapse-item>
          <el-collapse-item title="IP 宕机模板" name="2">
             <el-input type="textarea" :rows="3" v-model="form.template_ip_down" />
          </el-collapse-item>
          <el-collapse-item title="带宽告警模板" name="3">
             <el-input type="textarea" :rows="3" v-model="form.template_bw_limit" placeholder="Node {{node_id}} Bandwidth Exceeded" />
          </el-collapse-item>
        </el-collapse>
      </el-tab-pane>
      
      <el-tab-pane label="实时监控">
        <el-alert title="即将上线: 实时监控仪表盘" type="warning" :closable="false" />
      </el-tab-pane>
    </el-tabs>
    
    <div style="margin-top: 20px;">
      <el-button type="primary" @click="saveConfig">保存所有配置</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import request from '@/utils/request'
import { ElMessage } from 'element-plus'

const form = reactive({
    notify_email: '',
    notify_telegram: '',
    template_ip_up: '',
    template_ip_down: '',
    template_bw_limit: ''
})

const getConfig = () => {
    request.get('/monitor_config').then(res => {
        if(res.data) Object.assign(form, res.data)
    })
}

const saveConfig = () => {
    request.post('/monitor_config', form).then(() => {
        ElMessage.success('保存成功')
    })
}

onMounted(() => getConfig())
</script>
