<template>
  <div class="app-container">
    <el-card>
      <el-tabs v-model="activeTab">
        <el-tab-pane label="DNS API" name="dns">
          <div class="toolbar">
            <span></span>
            <el-button type="primary" @click="showAddDialog">
              <el-icon><Plus /></el-icon>
              ??DNS API
            </el-button>
          </div>

          <el-table :data="providers" style="width: 100%" v-loading="loading">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="name" label="??" width="200" />
            <el-table-column prop="type" label="???" width="150">
              <template #default="{ row }">
                <el-tag>{{ formatType(row.type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="????" />
            <el-table-column label="??" width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" @click="handleDelete(row)">??</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="CNAME ??" name="cname">
          <div class="filter-container">
            <div class="left">
              <el-button type="primary" @click="handleCreateCname">????</el-button>
              <el-button type="danger" :disabled="selectedCnames.length === 0" @click="handleBatchDeleteCname">??</el-button>
              <el-input
                v-model="cnameQuery.keyword"
                placeholder="??????"
                style="width: 200px; margin-left: 10px;"
                @keyup.enter="getCnameList"
              >
                <template #append>
                  <el-button :icon="Search" @click="getCnameList" />
                </template>
              </el-input>
            </div>
          </div>

          <el-table :data="cnameList" style="width: 100%" border @selection-change="handleCnameSelectionChange" v-loading="cnameLoading">
            <el-table-column type="selection" width="55" />
            <el-table-column prop="id" label="ID" width="80" align="center" />
            <el-table-column prop="domain" label="??" />
            <el-table-column prop="note" label="??" />
            <el-table-column label="??" width="150" align="center">
              <template #default="{ row }">
                <el-button link type="danger" @click="handleDeleteCname(row)">??</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog v-model="dialogVisible" title="?? DNS API" width="600px">
      <el-form :model="form" label-width="160px" ref="formRef">
        <el-form-item label="DNS ???" required>
          <el-select v-model="form.type" placeholder="???" @change="handleTypeChange" style="width: 100%">
            <el-option v-for="t in providerTypes" :key="t.type" :label="t.name + ' (' + t.type + ')'" :value="t.type" />
          </el-select>
        </el-form-item>

        <el-form-item label="??" required>
          <el-input v-model="form.name" placeholder="?????DNS??" />
        </el-form-item>

        <template v-if="currentTypeConfig">
          <el-form-item
            v-for="field in currentTypeConfig.fields"
            :key="field"
            :label="getDynamicLabel(form.type, field)"
            required
          >
            <el-input v-model="form.credentials[field]" :placeholder="'??? ' + getDynamicLabel(form.type, field)" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">??</el-button>
          <el-button type="primary" @click="submitForm" :loading="submitting">??</el-button>
        </span>
      </template>
    </el-dialog>

    <el-dialog v-model="cnameDialogVisible" title="??CNAME??" width="500px">
      <el-form :model="cnameForm" label-width="100px">
        <el-form-item label="??" required>
          <el-input v-model="cnameForm.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model="cnameForm.note" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="cnameDialogVisible = false">??</el-button>
          <el-button type="primary" @click="submitCnameForm">??</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, reactive } from 'vue'
import request from '@/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'

const activeTab = ref('dns')

const loading = ref(false)
const providers = ref([])
const providerTypes = ref([])
const dialogVisible = ref(false)
const submitting = ref(false)

const form = ref({
  name: '',
  type: '',
  credentials: {}
})

const labelMaps = {
  aliyun: { id: 'AccessKey ID', secret: 'AccessKey Secret' },
  huawei: { id: 'Access Key Id', secret: 'Secret Access Key' },
  dnsla: { id: 'APIID', secret: 'API ??' },
  dnspod: { id: 'ID', token: 'Token' },
  '51dns': { id: 'API Key', secret: 'API Secret' },
  cloudflare: { email: 'Email', key: 'API Key' }
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

const formatType = type => {
  const t = providerTypes.value.find(item => item.type === type)
  return t ? t.name : type
}

const loadData = () => {
  loading.value = true
  request.get('/dns/providers').then(res => {
    if (res.code === 0) {
      providers.value = res.data.list
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

const showAddDialog = () => {
  form.value = { name: '', type: '', credentials: {} }
  dialogVisible.value = true
}

const handleTypeChange = () => {
  form.value.credentials = {}
}

const submitForm = () => {
  if (!form.value.name || !form.value.type) {
    ElMessage.error('???????????')
    return
  }

  if (currentTypeConfig.value) {
    for (const field of currentTypeConfig.value.fields) {
      if (!form.value.credentials[field]) {
        ElMessage.error(`??? ${getDynamicLabel(form.value.type, field)}`)
        return
      }
    }
  }

  submitting.value = true
  request.post('/dns/providers', {
    name: form.value.name,
    type: form.value.type,
    credentials: JSON.stringify(form.value.credentials)
  }).then(res => {
    if (res.code === 0) {
      ElMessage.success('????')
      dialogVisible.value = false
      loadData()
    } else {
      ElMessage.error(res.msg || '????')
    }
  }).finally(() => {
    submitting.value = false
  })
}

const handleDelete = row => {
  ElMessageBox.confirm('?????DNS API?', '??', {
    confirmButtonText: '??',
    cancelButtonText: '??',
    type: 'warning'
  }).then(() => {
    request.delete(`/dns/providers/${row.id}`).then(res => {
      if (res.code === 0) {
        ElMessage.success('????')
        loadData()
      }
    })
  })
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

const submitCnameForm = () => {
  if (!cnameForm.domain) {
    ElMessage.error('?????')
    return
  }
  request.post('/cname_domains', cnameForm).then(res => {
    if (res.code === 0) {
      ElMessage.success('????')
      cnameDialogVisible.value = false
      getCnameList()
    } else {
      ElMessage.error(res.msg || '????')
    }
  })
}

const handleDeleteCname = row => {
  ElMessageBox.confirm('?????CNAME???', '??', {
    confirmButtonText: '??',
    cancelButtonText: '??',
    type: 'warning'
  }).then(() => {
    request.delete(`/cname_domains/${row.id}`).then(res => {
      if (res.code === 0) {
        ElMessage.success('????')
        getCnameList()
      }
    })
  })
}

const handleCnameSelectionChange = val => {
  selectedCnames.value = val
}

const handleBatchDeleteCname = () => {
  ElMessage.warning('?????????????')
}

onMounted(() => {
  loadData()
  loadTypes()
  getCnameList()
})
</script>

<style scoped>
.toolbar {
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
}

.filter-container {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}
</style>
