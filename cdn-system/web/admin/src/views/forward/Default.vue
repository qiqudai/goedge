<template>
  <div class="app-container">
    <el-card shadow="never" class="layout-card">
      <el-tabs v-model="activeTopTab" class="custom-tabs" @tab-change="handleTopTab">
        <el-tab-pane label="转发列表" name="list" />
        <el-tab-pane label="默认设置" name="default">
          <div class="filter-container">
            <el-button type="primary" @click="openCreate">新增设置</el-button>
          </div>

          <AppTable
            v-loading="loading"
            :data="settings"
            border
            fit
            highlight-current-row
            persist-key="forward-default-list"
            style="width: 100%;"
          >
            <el-table-column prop="key" label="设置项" width="180">
              <template #default="{ row }">
                {{ keyLabel(row.key) }}
              </template>
            </el-table-column>
            <el-table-column prop="value" label="设置值" min-width="180">
              <template #default="{ row }">
                {{ valueLabel(row.key, row.value) }}
              </template>
            </el-table-column>
            <el-table-column prop="scope" label="生效范围" width="140">
              <template #default="{ row }">
                {{ row.scope === 'group' ? '转发分组' : '全局' }}
              </template>
            </el-table-column>
            <el-table-column prop="group_name" label="转发分组" min-width="180" />
            <el-table-column label="操作" width="160" align="center" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" @click="removeSetting(row)">删除</el-button>
              </template>
            </el-table-column>
          </AppTable>
        </el-tab-pane>
        <el-tab-pane label="实时监控" name="monitor" />
      </el-tabs>
    </el-card>

    <el-dialog v-model="dialogVisible" title="新增设置" width="520px">
      <el-form :model="form" label-width="100px" style="padding-top: 10px;">
        <el-form-item label="设置项">
          <el-select v-model="form.key" placeholder="请选择" style="width: 100%;" @change="handleKeyChange">
            <el-option label="开启proxy_protocol" value="proxy_protocol" />
            <el-option label="监听协议" value="listen_protocol" />
            <el-option label="负载方式" value="balance_way" />
          </el-select>
        </el-form-item>
        <el-form-item label="设置值">
          <template v-if="form.key === 'proxy_protocol'">
            <el-radio-group v-model="form.value">
              <el-radio :value="true">是</el-radio>
              <el-radio :value="false">否</el-radio>
            </el-radio-group>
          </template>
          <template v-else-if="form.key === 'listen_protocol'">
            <el-radio-group v-model="form.value">
              <el-radio value="tcp">TCP</el-radio>
              <el-radio value="udp">UDP</el-radio>
            </el-radio-group>
          </template>
          <template v-else>
            <el-radio-group v-model="form.value">
              <el-radio value="rr">轮循</el-radio>
              <el-radio value="ip_hash">定源</el-radio>
            </el-radio-group>
          </template>
        </el-form-item>
        <el-form-item label="生效范围">
          <el-radio-group v-model="form.scope">
            <el-radio value="global">全局</el-radio>
            <el-radio value="group">转发分组</el-radio>
          </el-radio-group>
          <el-select v-if="form.scope === 'group'" v-model="form.group_id" placeholder="请选择" style="width: 220px; margin-left: 12px;">
            <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const router = useRouter()
const activeTopTab = ref('default')

const handleTopTab = (name) => {
  const map = {
    list: '/forward/list',
    default: '/forward/default',
    monitor: '/forward/monitor'
  }
  const path = map[name]
  if (name === 'default') {
    fetchSettings()
    loadGroups()
    return
  }
  if (path) router.push(path)
}

const settings = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const groupOptions = ref([])

const form = reactive({
  key: 'proxy_protocol',
  value: true,
  scope: 'global',
  group_id: ''
})

const fetchSettings = async () => {
  loading.value = true
  try {
    const res = await request.get('/forward_defaults')
    settings.value = res.list || []
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  const res = await request.get('/forward_groups')
  groupOptions.value = res.list || []
}

const keyLabel = key => {
  if (key === 'proxy_protocol') return '开启proxy_protocol'
  if (key === 'listen_protocol') return '监听协议'
  if (key === 'balance_way') return '负载方式'
  return key
}

const valueLabel = (key, value) => {
  if (key === 'proxy_protocol') return value ? '是' : '否'
  if (key === 'listen_protocol') return value === 'udp' ? 'UDP' : 'TCP'
  if (key === 'balance_way') return value === 'ip_hash' ? '定源' : '轮循'
  return value
}

const handleKeyChange = () => {
  if (form.key === 'proxy_protocol') {
    form.value = true
  } else if (form.key === 'listen_protocol') {
    form.value = 'tcp'
  } else {
    form.value = 'rr'
  }
}

const openCreate = () => {
  dialogVisible.value = true
  Object.assign(form, { key: 'proxy_protocol', value: true, scope: 'global', group_id: '' })
}

const submitForm = async () => {
  const payload = {
    key: form.key,
    value: form.value,
    scope: form.scope,
    group_id: form.scope === 'group' ? form.group_id : 0
  }
  await request.post('/forward_defaults', payload)
  ElMessage.success('新增成功')
  dialogVisible.value = false
  fetchSettings()
}

const removeSetting = row => {
  ElMessageBox.confirm('确认删除该设置?', '提示').then(async () => {
    await request.delete('/forward_defaults', { data: { id: row.id } })
    ElMessage.success('删除成功')
    fetchSettings()
  })
}

onMounted(() => {
  fetchSettings()
  loadGroups()
})
</script>

<style scoped>
.app-container { padding: 20px; }
.layout-card { border: none; }
.custom-tabs :deep(.el-tabs__item) { font-weight: 600; }
.filter-container { margin-bottom: 20px; }
</style>
