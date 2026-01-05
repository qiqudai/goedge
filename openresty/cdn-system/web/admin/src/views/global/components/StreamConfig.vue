<template>
  <el-form v-loading="loading" label-width="150px" class="config-form">
    <el-form-item label="监听协议" style="max-width: 600px;">
      <el-radio-group v-model="form.listenProtocol" @change="saveConfig">
        <el-radio value="tcp">tcp</el-radio>
        <el-radio value="udp">udp</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="负载方式" style="max-width: 600px;">
      <el-radio-group v-model="form.balanceWay" @change="saveConfig">
        <el-radio value="rr">轮循</el-radio>
        <el-radio value="ip_hash">定源</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="开启proxy protocol" style="max-width: 500px;">
      <el-switch v-model="form.proxyProtocol" @change="saveConfig" />
    </el-form-item>
  </el-form>
</template>

<script setup>
import { reactive, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const form = reactive({
  listenProtocol: 'tcp',
  balanceWay: 'rr',
  proxyProtocol: false
})

const loading = ref(false)

const saveConfig = async () => {
  try {
    const items = [
      { name: 'listen_protocol', value: form.listenProtocol },
      { name: 'balance_way', value: form.balanceWay },
      { name: 'proxy_protocol', value: form.proxyProtocol ? '1' : '0' }
    ]
    await request.post('/config_items', {
      type: 'stream_default_config',
      scope_name: 'global',
      scope_id: 0,
      items: items
    })
    ElMessage.success('保存成功')
  } catch (err) {
    ElMessage.error('保存失败')
  }
}

const loadConfig = async () => {
  const res = await request.get('/config_items', {
    params: { type: 'stream_default_config', scope_name: 'global', scope_id: 0 }
  })
  const list = res?.list || []
  list.forEach((item) => {
    switch (item.name) {
      case 'listen_protocol':
        form.listenProtocol = item.value
        break
      case 'balance_way':
        form.balanceWay = item.value
        break
      case 'proxy_protocol':
        form.proxyProtocol = (item.value === '1')
        break
    }
  })
}

onMounted(async () => {
  loading.value = true
  try {
    await loadConfig()
  } finally {
    loading.value = false
  }
})
</script>
