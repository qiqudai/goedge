<template>
  <el-form label-width="150px" class="config-form">
    <el-form-item label="默认证书类型" style="max-width: 600px;">
      <el-radio-group v-model="form.provider" @change="saveConfig">
        <el-radio value="zerossl">zerossl</el-radio>
        <el-radio value="lets">lets</el-radio>
        <el-radio value="buypass">buypass</el-radio>
        <el-radio value="google">google</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="DNS API" style="max-width: 500px;">
      <el-select v-model="form.dnsapiId" @change="saveConfig">
        <el-option
          v-for="item in dnsapis"
          :key="item.id"
          :label="item.name"
          :value="item.id" />
      </el-select>
      <div class="help-text">设置后，申请证书将使用此DNS API</div>
    </el-form-item>
  </el-form>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const form = reactive({
  provider: 'lets',
  dnsapiId: 0
})

const dnsapis = ref([])

const saveConfig = async () => {
  try {
    const items = [
      { name: 'cert_default_type', value: form.provider },
      { name: 'cert_default_dnsapi_id', value: String(form.dnsapiId || 0) }
    ]
    await request.post('/config_items', {
      type: 'cert_default_config',
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
    params: { type: 'cert_default_config', scope_name: 'global', scope_id: 0 }
  })
  const list = res?.list || []
  list.forEach((item) => {
    switch (item.name) {
      case 'cert_default_type':
        form.provider = item.value
        break
      case 'cert_default_dnsapi_id':
        form.dnsapiId = Number(item.value) || 0
        break
    }
  })
}

const loadDnsApis = async () => {
  const res = await request.get('/dnsapi')
  dnsapis.value = res?.data?.data?.list || []
}

onMounted(() => {
  loadConfig()
  loadDnsApis()
})
</script>
