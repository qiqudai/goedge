<template>
  <div class="dnsapi-toolbar">
    <el-button type="primary" @click="handleEdit()">新增 DNS 接口</el-button>
    <el-button :disabled="!selectedRows.length" @click="handleDeleteBatch">删除</el-button>
  </div>
  <AppTable
    v-loading="loading"
    :data="list"
    border
    persist-key="website-dnsapi-list"
    storage-key="website-dnsapi-list"
    @selection-change="(rows) => selectedRows = rows"
  >
    <el-table-column type="selection" width="55" align="center" />
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="name" label="名称" min-width="150" />
    <el-table-column prop="type" label="类型" width="120">
      <template #default="{ row }">
        {{ getTypeName(row.type) }}
      </template>
    </el-table-column>
    <el-table-column prop="remark" label="备注" min-width="150" />
    <el-table-column label="操作" width="120" align="center">
      <template #default="{ row }">
        <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
        <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
      </template>
    </el-table-column>
  </AppTable>

  <el-dialog v-model="visible" title="DNS 接口设置" width="600px">
    <el-form :model="form" label-width="120px">
      <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="类型" required>
        <el-select v-model="form.type" style="width: 100%;" @change="handleTypeChange">
          <el-option v-for="t in types" :key="t.type" :label="t.name" :value="t.type" />
        </el-select>
      </el-form-item>
      
      <!-- Dynamic Authentication Fields -->
      <template v-if="currentType">
        <el-form-item 
          v-for="field in currentType.fields" 
          :key="field"
          :label="formatFieldLabel(form.type, field)"
          required
        >
          <el-input v-model="form.auth[field]" :placeholder="'请输入 ' + formatFieldLabel(form.type, field)" />
        </el-form-item>
      </template>

      <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false" :disabled="submitting">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'

import { DNS_PROVIDERS, DNS_API_FIELD_LABELS } from '@/constants/dns'

const list = ref([])
const types = ref([])
const loading = ref(false)
const submitting = ref(false)
const selectedRows = ref([])
const visible = ref(false)
const form = reactive({ id: 0, name: '', type: '', remark: '', auth: {} })

const currentType = computed(() => {
  return types.value.find(t => t.type === form.type)
})

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


const getTypeName = (type) => {
  const t = types.value.find(item => item.type === type)
  return t ? t.name : (DNS_PROVIDERS[type]?.name || type)
}

const formatFieldLabel = (type, field) => {
  const typeConfig = DNS_API_FIELD_LABELS[type]
  if (typeConfig && typeConfig[field]) {
    return translateLabel(typeConfig[field])
  }
  return translateLabel(field)
}


// Ensure form.auth properties exist for reactivity
const handleTypeChange = () => {
  form.auth = {}
  if (currentType.value && currentType.value.fields) {
    currentType.value.fields.forEach(f => {
      form.auth[f] = ''
    })
  }
}

const fetchData = async () => {
  loading.value = true
  const res = await request.get('/dnsapi')
  list.value = res.data?.list || res.list || []
  loading.value = false
}

const fetchTypes = async () => {
  try {
    const [providerRes, dnsapiRes] = await Promise.all([
      request.get('/dns/providers/types'),
      request.get('/dnsapi/types')
    ])
    const providerTypes = providerRes.data?.types || providerRes.types || []
    const allow = new Set(providerTypes.map(item => item.type))
    const dnsapiTypes = dnsapiRes.data?.types || dnsapiRes.types || []
    types.value = allow.size > 0 ? dnsapiTypes.filter(item => allow.has(item.type)) : dnsapiTypes
  } catch (e) {
    const res = await request.get('/dnsapi/types')
    types.value = res.data?.types || res.types || []
  }
}

const handleEdit = (row) => {
  if (row) {
    let auth = {}
    try {
      auth = JSON.parse(row.auth)
    } catch(e) {
      auth = {}
    }
    Object.assign(form, { ...row, auth })
  } else {
    Object.assign(form, { id: 0, name: '', type: '', remark: '', auth: {} })
  }
  visible.value = true
}

const handleSubmit = async () => {
  if (submitting.value) return
  if (!form.name || !form.type) {
    ElMessage.error('请填写必要信息')
    return
  }
  const payload = { ...form, auth: JSON.stringify(form.auth) }
  submitting.value = true
  try {
    if (form.id) {
      await request.put(`/dnsapi/${form.id}`, payload)
    } else {
      await request.post('/dnsapi', payload)
    }
    ElMessage.success('保存成功')
    visible.value = false
    fetchData()
  } finally {
    submitting.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除?', '提示').then(async () => {
    await request.delete(`/dnsapi/${row.id}`)
    fetchData()
  })
}

const handleDeleteBatch = () => {
  if (!selectedRows.value.length) return
  ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 个 DNS 接口?`, '提示').then(async () => {
    // Assuming backend supports batch delete or we loop
    // Since we don't have batch endpoint confirmed, we'll try loop
    for (const row of selectedRows.value) {
       await request.delete(`/dnsapi/${row.id}`)
    }
    ElMessage.success('批量删除成功')
    fetchData()
    selectedRows.value = []
  })
}

onMounted(() => {
  fetchData()
  fetchTypes()
})
</script>

<style scoped>
.dnsapi-toolbar { margin-bottom: 20px; }
</style>
