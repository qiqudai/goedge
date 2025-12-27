<template>
  <div class="app-container">
    <el-card>
      <el-tabs v-model="activeTab">
        <el-tab-pane label="DNS配置" name="dns">
          <el-form :model="form" label-width="120px" class="dns-form">
            <el-form-item label="DNS提供商" required>
              <el-select v-model="form.type" placeholder="请选择" @change="handleTypeChange" style="width: 320px;">
                <el-option v-for="t in providerTypes" :key="t.type" :label="getProviderLabel(t)" :value="t.type" />
              </el-select>
            </el-form-item>

            <template v-if="currentTypeConfig">
              <el-form-item
                v-for="field in currentTypeConfig.fields"
                :key="field"
                :label="getDynamicLabel(form.type, field)"
                required
              >
                <el-input v-model="form.credentials[field]" :placeholder="'请输入' + getDynamicLabel(form.type, field)" style="width: 320px;" />
              </el-form-item>
            </template>

            <el-form-item label="TTL">
              <el-input-number v-model="form.ttl" :min="1" />
            </el-form-item>
            <el-form-item label="开启IP权重">
              <el-switch v-model="form.ip_weight" />
            </el-form-item>
            <el-form-item label="DNS错误">
              <span class="dns-error">{ dnsError }</span>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="submitForm" :loading="submitting">保存</el-button>
              <el-button @click="handleFixRecords">记录修复</el-button>
              <el-button @click="handleClearInvalid">清除CDN无关解析</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="CNAME 域名" name="cname">
          <div class="filter-container">
            <div class="left">
              <el-button type="primary" @click="handleCreateCname">添加域名</el-button>
              <el-button type="danger" :disabled="selectedCnames.length === 0" @click="handleBatchDeleteCname">批量删除</el-button>
              <el-input
                v-model="cnameQuery.keyword"
                placeholder="搜索域名"
                style="width: 200px; margin-left: 10px;"
                @keyup.enter="getCnameList"
              >
                <template #append>
                  <el-button :icon="Search" @click="getCnameList" />
                </template>
              </el-input>
            </div>
          </div>

          <AppTable
            :data="cnameList"
            :loading="cnameLoading"
            persist-key="cname"
            style="width: 100%"
            border
            @selection-change="handleCnameSelectionChange"
          >
            <el-table-column type="selection" width="55" />
            <el-table-column prop="id" label="ID" width="80" align="center" />
            <el-table-column prop="domain" label="域名" />
            <el-table-column prop="note" label="备注" />
            <el-table-column label="操作" width="150" align="center">
              <template #default="{ row }">
                <el-button link type="danger" @click="handleDeleteCname(row)">删除</el-button>
              </template>
            </el-table-column>
          </AppTable>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    

    <el-dialog v-model="cnameDialogVisible" title="添加CNAME域名" width="500px">
      <el-form :model="cnameForm" label-width="100px">
        <el-form-item label="域名" required>
          <el-input v-model="cnameForm.domain" placeholder="example.com" @blur="handleCnameBlur" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="cnameForm.note" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="cnameDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitCnameForm">提交</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, reactive, watch } from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'

const activeTab = ref('dns')

const loading = ref(false)
const providers = ref([])
const providerTypes = ref([])
const providerLabelMap = {
  aliyun: 'Aliyun (aliyun.com,alidns.aliyun.com)',
  huawei: 'Huawei (huaweicloud.com)',
  dnsla: 'DNSLA (dns.la)',
  dnspod: 'DNSPod (dnspod.cn)',
  dnspod_intl: 'DNSPod Intl (dnspod.com)',
  '51dns': '51DNS (51dns.com)',
  cloudflare: 'Cloudflare (cloudflare.com)',
  godaddy: 'GoDaddy (godaddy.com)'
}

const submitting = ref(false)

const currentProviderId = ref(0)
const dnsError = ref('没有错误')

const form = ref({
  name: '',
  type: '',
  credentials: {},
  ttl: 600,
  ip_weight: false
})

const labelMaps = {
  aliyun: { access_key_id: 'AccessKey ID', access_key_secret: 'AccessKey Secret' },
  huawei: { id: 'Access Key Id', secret: 'Secret Access Key' },
  dnsla: { id: 'APIID', secret: 'API Key' },
  dnspod: { id: 'ID', token: 'Token' },
  dnspod_intl: { id: 'ID', token: 'Token' },
  '51dns': { id: 'API Key', secret: 'API Secret' },
  cloudflare: { email: 'Email', api_key: 'API Key' },
  godaddy: { key: 'Key', secret: 'Secret' }
}

const getDynamicLabel = (type, field) => {
  if (labelMaps[type] && labelMaps[type][field]) {
    return labelMaps[type][field]
  }
  return field.replace(/_/g, ' ').toUpperCase()
}

const currentTypeConfig = computed(() => {
  return providerTypes.value.find(t => t.type === form.value.type)
})

const getProviderLabel = item => {
  if (!item) {
    return ''
  }
  return providerLabelMap[item.type] || item.name || item.type
}

const applyProvider = item => {
  currentProviderId.value = item.id || 0
  form.value.type = item.type || ''
  form.value.name = item.name || ''
  let auth = {}
  if (item.auth) {
    try {
      auth = JSON.parse(item.auth)
    } catch (e) {
      auth = {}
    }
  }
  const normalized = { ...auth }
  const ttl = normalized.ttl
  const ipWeight = normalized.ip_weight
  delete normalized.ttl
  delete normalized.ip_weight
  form.value.credentials = normalized
  form.value.ttl = typeof ttl === 'number' ? ttl : 600
  form.value.ip_weight = !!ipWeight
}

const loadProviders = () => {
  loading.value = true
  request.get('/dns/providers').then(res => {
    if (res.code === 0) {
      const list = res.data.list || []
      providers.value = list
      if (list.length > 0) {
        applyProvider(list[0])
      } else {
        currentProviderId.value = 0
      }
    }
  }).finally(() => {
    loading.value = false
  })
}

const loadTypes = () => {
  request.get('/dns/providers/types').then(res => {
    if (res.code === 0) {
      providerTypes.value = res.data.types
    }
  })
}

const handleTypeChange = () => {
  const match = providers.value.find(item => item.type === form.value.type)
  if (match) {
    applyProvider(match)
    return
  }
  currentProviderId.value = 0
  form.value.credentials = {}
  form.value.ttl = 600
  form.value.ip_weight = false
}

const submitForm = () => {
  if (!form.value.type) {
    ElMessage.error('请选择DNS提供商')
    return
  }

  if (currentTypeConfig.value) {
    for (const field of currentTypeConfig.value.fields) {
      if (!form.value.credentials[field]) {
        ElMessage.error(`请输入${getDynamicLabel(form.value.type, field)}`)
        return
      }
    }
  }

  const payload = {
    name: getProviderLabel(currentTypeConfig.value) || form.value.type,
    type: form.value.type,
    credentials: JSON.stringify({
      ...form.value.credentials,
      ttl: form.value.ttl,
      ip_weight: form.value.ip_weight
    })
  }

  submitting.value = true
  const createProvider = () => {
    return request.post('/dns/providers', payload).then(res => {
      if (res.code === 0) {
        ElMessage.success('保存成功')
        loadProviders()
      } else {
        ElMessage.error(res.msg || '操作失败')
      }
    }).finally(() => {
      submitting.value = false
    })
  }

  if (currentProviderId.value > 0) {
    request.delete(`/dns/providers/${currentProviderId.value}`).then(() => {
      createProvider()
    }).catch(() => {
      submitting.value = false
    })
    return
  }
  createProvider()
}

const cnameList = ref([])
const cnameLoading = ref(false)
const cnameDialogVisible = ref(false)
const cnameForm = reactive({ domain: '', note: '' })
const cnameQuery = reactive({ keyword: '' })
const selectedCnames = ref([])

const getCnameList = () => {
  cnameLoading.value = true
  request.get('/cname_domains').then(res => {
    if (res.code === 0) {
      let list = res.data.list || []
      if (cnameQuery.keyword) {
        list = list.filter(item => item.domain.includes(cnameQuery.keyword))
      }
      cnameList.value = list
    }
  }).finally(() => {
    cnameLoading.value = false
  })
}

const handleCreateCname = () => {
  cnameForm.domain = ''
  cnameForm.note = ''
  cnameDialogVisible.value = true
}

const normalizeDomainInput = value => {
  let v = (value || '').toString().trim().toLowerCase()
  if (v.startsWith('http://')) {
    v = v.slice(7)
  } else if (v.startsWith('https://')) {
    v = v.slice(8)
  }
  const slashIndex = v.indexOf('/')
  if (slashIndex != -1) {
    v = v.slice(0, slashIndex)
  }
  const hashIndex = v.indexOf('#')
  if (hashIndex != -1) {
    v = v.slice(0, hashIndex)
  }
  const queryIndex = v.indexOf('?')
  if (queryIndex != -1) {
    v = v.slice(0, queryIndex)
  }
  const colonIndex = v.indexOf(':')
  if (colonIndex != -1) {
    v = v.slice(0, colonIndex)
  }
  v = v.replace(/\.+$/g, '')
  return v
}

const isValidDomain = value => {
  if (!value || value.length > 253) {
    return false
  }
  const parts = value.split('.')
  if (parts.length < 2) {
    return false
  }
  for (const part of parts) {
    if (!part || part.length > 63) {
      return false
    }
    if (part.startsWith('-') || part.endsWith('-')) {
      return false
    }
    if (!/^[a-z0-9-]+$/.test(part)) {
      return false
    }
  }
  return true
}

const handleCnameBlur = () => {
  cnameForm.domain = normalizeDomainInput(cnameForm.domain)
}

const handleFixRecords = () => {
  ElMessage.info('功能开发中')
}

const handleClearInvalid = () => {
  ElMessage.info('功能开发中')
}

const submitCnameForm = () => {
  const normalized = normalizeDomainInput(cnameForm.domain)
  if (!normalized) {
    ElMessage.error('请输入域名')
    return
  }
  if (!isValidDomain(normalized)) {
    ElMessage.error('域名格式不正确')
    return
  }
  cnameForm.domain = normalized
  request.post('/cname_domains', cnameForm).then(res => {
    if (res.code === 0) {
      ElMessage.success('添加成功')
      cnameDialogVisible.value = false
      getCnameList()
    } else {
      ElMessage.error(res.msg || '操作失败')
    }
  })
}

const handleDeleteCname = row => {
  ElMessageBox.confirm('确认删除该CNAME域名吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.delete(`/cname_domains/${row.id}`).then(res => {
      if (res.code === 0) {
        ElMessage.success('已删除')
        getCnameList()
      }
    })
  })
}

const handleCnameSelectionChange = val => {
  selectedCnames.value = val
}


watch(
  () => form.value.type,
  () => {
    handleTypeChange()
  }
)

const handleBatchDeleteCname = () => {
  ElMessage.warning('批量删除功能暂未开放')
}

onMounted(() => {
  loadProviders()
  loadTypes()
  getCnameList()
})
</script>

<style scoped>
.filter-container {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.dns-form {
  max-width: 560px;
}

.dns-error {
  color: #999;
}
</style>
