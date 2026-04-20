<template>
  <div class="dnsapi-section">
    <div class="filter-container">
      <el-select
        v-if="isAdmin"
        v-model="selectedUserId"
        placeholder="请选择用户"
        style="width: 280px"
        filterable
        clearable
        :loading="userLoading"
        @visible-change="handleUserDropdown"
        @change="handleUserChange"
      >
        <el-option
          v-for="u in users"
          :key="u.id"
          :label="`${u.name} (${u.username || u.email || u.id})`"
          :value="u.id"
        />
      </el-select>
      <el-button type="primary" @click="openDnsapiDialog">新增 DNS 接口</el-button>
      <el-button :disabled="!selectedDnsapi.length" @click="removeDnsapiBatch">删除</el-button>
    </div>
    <AppTable
      v-loading="dnsapiLoading"
      :data="dnsapiList"
      border
      style="width: 100%;"
      persist-key="dnsapi-list"
      storage-key="dnsapi-list"
      :show-pagination="false"
      @selection-change="handleDnsapiSelection"
    >
      <el-table-column type="selection" width="55" />
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column prop="type" label="DNS" width="140">
        <template #default="{ row }">
          <el-tag>{{ formatDnsType(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="remark" label="备注" min-width="200" show-overflow-tooltip />
      <el-table-column label="操作" width="140" align="center">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="openDnsapiEdit(row)">编辑</el-button>
          <el-button link type="danger" size="small" @click="removeDnsapi(row)">删除</el-button>
        </template>
      </el-table-column>
    </AppTable>
  </div>

  <el-dialog v-model="dnsapiDialogVisible" title="新增 DNS 接口" width="520px">
    <el-form :model="dnsapiForm" label-width="90px">
      <el-form-item v-if="isAdmin" label="所属用户" required>
        <el-select
          v-model="dnsapiForm.user_id"
          placeholder="请选择用户"
          style="width: 100%;"
          filterable
          clearable
          :loading="userLoading"
          @visible-change="handleUserDropdown"
        >
          <el-option
            v-for="u in users"
            :key="u.id"
            :label="`${u.name} (${u.username || u.email || u.id})`"
            :value="u.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="名称" required>
        <el-input v-model="dnsapiForm.name" placeholder="请输入名称" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="dnsapiForm.remark" placeholder="请输入备注" />
      </el-form-item>
      <el-form-item label="DNS" required>
        <el-select v-model="dnsapiForm.type" placeholder="请选择" style="width: 100%;" @change="resetDnsapiAuth">
          <el-option v-for="t in dnsapiTypes" :key="t.type" :label="t.name" :value="t.type" />
        </el-select>
      </el-form-item>
      <el-form-item label="验证信息" v-if="currentDnsapiType">
        <div class="dnsapi-fields">
          <el-form-item
            v-for="field in currentDnsapiType.fields"
            :key="field"
            :label="dnsapiFieldLabel(dnsapiForm.type, field)"
            label-width="120px"
          >
            <el-input v-model="dnsapiForm.credentials[field]" />
          </el-form-item>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dnsapiDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitDnsapi">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const props = defineProps({
  isAdmin: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['list-updated'])

const dnsapiList = ref([])
const dnsapiLoading = ref(false)
const selectedDnsapi = ref([])
const dnsapiTypes = ref([])
const dnsapiDialogVisible = ref(false)
const users = ref([])
const userLoading = ref(false)
const selectedUserId = ref('')
const dnsapiForm = reactive({
  id: 0,
  user_id: '',
  name: '',
  remark: '',
  type: '',
  credentials: {}
})

const currentDnsapiType = computed(() => dnsapiTypes.value.find(t => t.type === dnsapiForm.type))

const dnsapiFieldLabel = (type, field) => {
  const mapping = {
    aliyun: { id: 'AccessKey ID（访问密钥ID）', secret: 'AccessKey Secret（访问密钥密码）' },
    huawei: { id: 'Access Key ID（访问密钥ID）', secret: 'Secret Access Key（访问密钥密码）' },
    dnsla: { id: 'API ID', secret: 'API 密钥' },
    dnspod: { id: 'ID', token: '令牌' },
    dnspod_intl: { secret_id: 'SecretId（密钥ID）', secret_key: 'SecretKey（密钥密码）' },
    cloudflare: { email: '邮箱', key: 'API 密钥' },
    godaddy: { key: '密钥', secret: '密钥密码' }
  }
  if (mapping[type] && mapping[type][field]) {
    return mapping[type][field]
  }
  return field.toUpperCase()
}

const formatDnsType = type => {
  const t = dnsapiTypes.value.find(item => item.type === type)
  return t ? t.name : type
}

const searchUsers = async (keyword = '') => {
  if (!props.isAdmin) return
  userLoading.value = true
  try {
    const res = await request.get('/users', { params: { keyword, pageSize: 200 } })
    users.value = (res.data?.list || res.list || []).map(item => ({
      ...item,
      username: item.username || item.email || item.name
    }))
  } finally {
    userLoading.value = false
  }
}

const handleUserDropdown = visible => {
  if (visible && props.isAdmin && !users.value.length) {
    searchUsers('')
  }
}

const handleUserChange = userId => {
  selectedUserId.value = userId || ''
  dnsapiList.value = []
  selectedDnsapi.value = []
  loadDnsapiList()
}

const loadDnsapiList = () => {
  if (props.isAdmin && !selectedUserId.value) {
    dnsapiList.value = []
    emit('list-updated', [])
    return
  }
  dnsapiLoading.value = true
  const params = {}
  if (props.isAdmin && selectedUserId.value) {
    params.user_id = Number(selectedUserId.value)
  }
  request.get('/dnsapi', { params }).then(res => {
    dnsapiList.value = res.data?.list || res.list || []
    emit('list-updated', dnsapiList.value)
    dnsapiLoading.value = false
  }).catch(() => {
    dnsapiLoading.value = false
  })
}

const loadDnsapiTypes = () => {
  Promise.all([
    request.get('/dns/providers/types'),
    request.get('/dnsapi/types')
  ]).then(([providerRes, dnsapiRes]) => {
    const providerTypes = providerRes.data?.types || providerRes.types || []
    const allow = new Set(providerTypes.map(item => item.type))
    const allTypes = dnsapiRes.data?.types || dnsapiRes.types || []
    dnsapiTypes.value = allow.size > 0 ? allTypes.filter(item => allow.has(item.type)) : allTypes
  }).catch(() => {
    request.get('/dnsapi/types').then(res => {
      dnsapiTypes.value = res.data?.types || res.types || []
    })
  })
}

const resetDnsapiAuth = () => {
  dnsapiForm.credentials = {}
}

const openDnsapiDialog = () => {
  if (props.isAdmin && !selectedUserId.value) {
    ElMessage.warning('请先选择用户')
    return
  }
  dnsapiForm.id = 0
  dnsapiForm.user_id = props.isAdmin ? Number(selectedUserId.value) || '' : ''
  dnsapiForm.name = ''
  dnsapiForm.remark = ''
  dnsapiForm.type = ''
  dnsapiForm.credentials = {}
  dnsapiDialogVisible.value = true
}

const openDnsapiEdit = row => {
  dnsapiForm.id = row.id
  dnsapiForm.user_id = Number(row.user_id || row.uid || selectedUserId.value) || ''
  dnsapiForm.name = row.name
  dnsapiForm.remark = row.remark || ''
  dnsapiForm.type = row.type
  dnsapiForm.credentials = row.auth ? JSON.parse(row.auth) : {}
  dnsapiDialogVisible.value = true
}

const submitDnsapi = () => {
  if (!dnsapiForm.name || !dnsapiForm.type) {
    ElMessage.warning('请填写完整信息')
    return
  }
  if (props.isAdmin && !dnsapiForm.user_id) {
    ElMessage.warning('请选择用户')
    return
  }
  const payload = {
    user_id: props.isAdmin ? Number(dnsapiForm.user_id) || 0 : undefined,
    name: dnsapiForm.name,
    remark: dnsapiForm.remark,
    type: dnsapiForm.type,
    auth: JSON.stringify(dnsapiForm.credentials || {})
  }
  if (dnsapiForm.id) {
    request.put(`/dnsapi/${dnsapiForm.id}`, payload).then(() => {
      ElMessage.success('更新成功')
      dnsapiDialogVisible.value = false
      loadDnsapiList()
    })
  } else {
    request.post('/dnsapi', payload).then(() => {
      ElMessage.success('创建成功')
      dnsapiDialogVisible.value = false
      loadDnsapiList()
    })
  }
}

const removeDnsapi = row => {
  ElMessageBox.confirm('确认删除该 DNS 接口?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    request.delete(`/dnsapi/${row.id}`).then(() => {
      ElMessage.success('删除成功')
      loadDnsapiList()
    })
  })
}

const handleDnsapiSelection = rows => {
  selectedDnsapi.value = rows
}

const removeDnsapiBatch = () => {
  if (!selectedDnsapi.value.length) return
  const ids = selectedDnsapi.value.map(row => row.id)
  ElMessageBox.confirm('确定删除选中的 DNS 接口?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    Promise.all(ids.map(id => request.delete(`/dnsapi/${id}`))).then(() => {
      ElMessage.success('删除成功')
      loadDnsapiList()
    })
  })
}

onMounted(() => {
  if (props.isAdmin) {
    searchUsers('')
  }
  loadDnsapiList()
  loadDnsapiTypes()
})
</script>

<style scoped>
.filter-container {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
</style>
