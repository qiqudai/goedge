<template>
  <el-form v-loading="loading" label-width="150px" class="config-form">
    <el-form-item label="默认证书类型" style="max-width: 600px;">
      <el-radio-group v-model="form.provider" @change="saveConfig">
        <el-radio value="zerossl">ZeroSSL</el-radio>
        <el-radio value="lets">Let's Encrypt</el-radio>
        <el-radio value="buypass">BuyPass</el-radio>
        <el-radio value="google">Google CA</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="DNS 接口" style="max-width: 500px;">
      <el-select v-model="form.dnsapiType" placeholder="选择 DNS 服务商" @change="handleDnsTypeChange">
        <el-option
          v-for="(item, key) in DNS_PROVIDERS"
          :key="key"
          :label="item.name"
          :value="key" />
      </el-select>
      <div class="help-text">设置后，申请证书将使用此 DNS 接口</div>
    </el-form-item>

    <template v-if="form.dnsapiType && DNS_API_FIELD_LABELS[form.dnsapiType]">
      <el-form-item
        v-for="(label, field) in DNS_API_FIELD_LABELS[form.dnsapiType]"
        :key="field"
        :label="translateLabel(label)"
        style="max-width: 500px;"
      >
        <el-input v-model="form.dnsapiData[field]" @change="saveConfig" />
      </el-form-item>
    </template>
  </el-form>
</template>

<script setup>
import { reactive, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import { DNS_PROVIDERS, DNS_API_FIELD_LABELS } from '@/constants/dns'

const labelTranslations = {
  'AccessKey ID': 'AccessKey ID（访问密钥ID）',
  'AccessKey Secret': 'AccessKey Secret（访问密钥密码）',
  'Access Key ID': 'Access Key ID（访问密钥ID）',
  'Secret Access Key': 'Secret Access Key（访问密钥密码）',
  'Access Key': 'Access Key（访问密钥）',
  'Secret Key': 'Secret Key（密钥密码）',
  'API ID': 'API ID',
  'API Password': 'API 密码',
  'API Key': 'API 密钥',
  'API Secret': 'API 密钥密码',
  'API Token': 'API 令牌',
  Token: '令牌',
  Email: '邮箱',
  Username: '用户名',
  User: '用户',
  'Client IP': '客户端 IP',
  'App ID': '应用 ID',
  'App Secret': '应用密钥',
  'Auth ID': '认证 ID',
  'Auth Password': '认证密码',
  SecretId: 'SecretId（密钥ID）',
  SecretKey: 'SecretKey（密钥密码）',
  ID: 'ID'
}

const translateLabel = (label) => labelTranslations[label] || label

const form = reactive({
  provider: 'lets',
  dnsapiType: '',
  dnsapiData: {}
})

const loading = ref(false)

const saveConfig = async () => {
  try {
    const items = [
      { name: 'cert_default_type', value: form.provider },
      { name: 'cert_default_dnsapi_type', value: form.dnsapiType },
      { name: 'cert_default_dnsapi_data', value: JSON.stringify(form.dnsapiData) }
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
  const list = res?.data?.list || res?.list || []
  list.forEach((item) => {
    switch (item.name) {
      case 'cert_default_type':
        form.provider = item.value
        break
      case 'cert_default_dnsapi_type':
        form.dnsapiType = item.value
        break
      case 'cert_default_dnsapi_data':
        try {
            form.dnsapiData = JSON.parse(item.value) || {}
        } catch(e) {
            form.dnsapiData = {}
        }
        break
    }
  })
}

const handleDnsTypeChange = () => {
    form.dnsapiData = {}
    saveConfig()
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

