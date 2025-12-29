<template>
  <div class="default-settings-container">
    <div class="toolbar">
      <el-button type="primary" :icon="Plus" @click="handleAdd">新增默认设置</el-button>
      <el-button type="danger" :icon="Delete" :disabled="!selectedRows.length" @click="handleBatchDelete">删除</el-button>
    </div>

    <AppTable
      v-loading="loading"
      :data="formattedList"
      border
      fit
      highlight-current-row
      style="width: 100%; margin-top: 10px;"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" align="center" />
      <el-table-column label="设置项" min-width="150" prop="label" sortable />
      <el-table-column label="设置值" min-width="200">
        <template #default="{ row }">
           <span class="value-text" :title="row.value" style="display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;">
              {{ row.value }}
           </span>
        </template>
      </el-table-column>
      <el-table-column label="范围" width="150" prop="scopeLabel" sortable />
      <el-table-column label="操作" width="120" align="center">
        <template #default="{ row }">
          <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </AppTable>

    <el-dialog
      v-model="visible"
      :title="mode === 'edit' ? '编辑默认设置' : '新增默认设置'"
      width="600px"
      append-to-body
      :destroy-on-close="true"
      :close-on-click-modal="false"
    >
      <el-form :model="form" label-width="120px" ref="formRef">
        <!-- Scope Selection -->
        <el-form-item label="设置项" required>
            <el-select 
                v-model="form.name" 
                filterable 
                :disabled="mode === 'edit'" 
                style="width: 100%" 
                @change="handleNameChange"
            >
                <el-option 
                    v-for="opt in defaultOptions" 
                    :key="opt.value" 
                    :label="opt.label" 
                    :value="opt.value" 
                />
            </el-select>
        </el-form-item>

        <el-form-item label="适用范围" required>
           <el-radio-group v-model="form.scope">
              <el-radio value="global">全局</el-radio>
              <el-radio value="group">分组</el-radio>
           </el-radio-group>
        </el-form-item>

        <el-form-item v-if="isAdmin" label="所属用户">
           <el-select 
                v-model="form.user_id" 
                filterable 
                remote 
                :remote-method="searchUsers" 
                :loading="userLoading" 
                placeholder="搜索用户" 
                style="width: 100%"
                clearable
            >
                <el-option v-for="u in userOptions" :key="u.id" :label="u.name" :value="u.id" />
           </el-select>
           <div class="form-tip">管理员必须指定用户</div>
        </el-form-item>

        <el-form-item v-if="form.scope === 'group'" label="网站分组" required>
           <el-select v-model="form.group_id" placeholder="选择分组" style="width: 100%" filterable>
                <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
           </el-select>
        </el-form-item>

        <!-- Dynamic Value Input -->
        <el-form-item label="设置值" required>
            <!-- Boolean (Switch) -->
            <el-switch v-if="currentType === 'bool'" v-model="form.boolValue" />

            <!-- Number -->
            <el-input-number v-else-if="currentType === 'number'" v-model="form.value" style="width: 100%;" />

            <!-- Select -->
            <el-select v-else-if="currentType === 'select'" v-model="form.selectValue" style="width: 100%;">
                <el-option v-for="opt in currentChoices" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            
            <!-- Multi Select -->
             <el-select v-else-if="currentType === 'multi'" v-model="form.multiValue" multiple style="width: 100%;">
                <el-option v-for="opt in currentChoices" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>

            <!-- Lines (Textarea) -->
            <el-input v-else-if="currentType === 'lines'" v-model="form.textValue" type="textarea" :rows="4" placeholder="每行一个值" />

             <!-- Region -->
            <div v-else-if="currentType === 'region'" style="width: 100%;">
                <CountrySelector v-model="form.region_custom" />
            </div>

            <!-- Headers -->
            <div v-else-if="currentType === 'headers'" style="width: 100%;">
                <div v-for="(h, idx) in form.headers" :key="idx" style="display: flex; gap: 8px; margin-bottom: 8px;">
                    <el-input v-model="h.name" placeholder="Name" style="flex: 1" />
                    <el-input v-model="h.value" placeholder="Value" style="flex: 1" />
                    <el-button type="danger" :icon="Minus" circle size="small" @click="removeHeader(idx)" />
                </div>
                <el-button type="primary" size="small" plain @click="addHeader">+ 添加Header</el-button>
            </div>
            
             <!-- Default Logic (Text) -->
            <el-input v-else v-model="form.textValue" />
        </el-form-item>

      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { Plus, Delete, Minus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'
import CountrySelector from '@/components/CountrySelector.vue'

const props = defineProps({
  isAdmin: Boolean
})

const loading = ref(false)
const list = ref([])
const selectedRows = ref([])
const visible = ref(false)
const mode = ref('create')

// Options Cache
const userOptions = ref([])
const userLoading = ref(false)
const groupOptions = ref([])
const dnsOptions = ref([]) // For dns_provider_id
const ccRuleOptions = ref([]) // For cc_rules

const formRef = ref(null)
const editScope = reactive({ name: '', id: 0, originalName: '' }) // Track original scope for put

const form = reactive({
  user_id: undefined,
  name: '',
  value: 0,
  boolValue: false,
  selectValue: '',
  multiValue: [],
  textValue: '',
  headers: [],
  region_mode: 'none',
  region_custom: [],
  scope: 'global',
  group_id: 0
})

// --- Logic from List-old.vue ---

const defaultOptions = [
  { label: '默认CC规则', value: 'cc_default_rule', type: 'select', choicesKey: 'cc_rules' },
  { label: '黑名单时间', value: 'security_black_time', type: 'number' },
  { label: '白名单时间', value: 'security_white_time', type: 'number' },
  { label: '搜索引擎爬虫', value: 'security_bot', type: 'select', choices: [
    { label: '不设置', value: 'none' },
    { label: '放行', value: 'allow' },
    { label: '拦截', value: 'deny' }
  ] },
  { label: '黑名单IP', value: 'black_ip', type: 'lines' },
  { label: '白名单IP', value: 'white_ip', type: 'lines' },
  { label: '屏蔽透明代理', value: 'security_shield_proxy', type: 'bool' },
  { label: '区域屏蔽', value: 'block_region', type: 'region' },
  { label: 'DNS API(解析)', value: 'dns_provider_id', type: 'select', choicesKey: 'dns_providers' },
  { label: 'HTTP监听端口', value: 'http_listen-port', type: 'number' },
  { label: 'HTTPS监听端口', value: 'https_listen-port', type: 'number' },
  { label: '强制HTTPS', value: 'https_listen-force_ssl_enable', type: 'bool' },
  { label: '开启HSTS', value: 'https_listen-hsts', type: 'bool' },
  { label: '开启HTTP2', value: 'https_listen-http2', type: 'bool' },
  { label: '开启HTTP3', value: 'https_listen-http3', type: 'bool' },
  { label: 'ssl_protocols', value: 'https_listen-ssl_protocols', type: 'multi', choices: [
    { label: 'SSLv2', value: 'SSLv2' },
    { label: 'SSLv3', value: 'SSLv3' },
    { label: 'TLSv1', value: 'TLSv1' },
    { label: 'TLSv1.1', value: 'TLSv1.1' },
    { label: 'TLSv1.2', value: 'TLSv1.2' },
    { label: 'TLSv1.3', value: 'TLSv1.3' }
  ] },
  { label: 'ssl_ciphers', value: 'https_listen-ssl_ciphers', type: 'text' },
  { label: 'ssl_prefer_server_ciphers', value: 'https_listen-ssl_prefer_server_ciphers', type: 'select', choices: [
    { label: 'On', value: 'on' },
    { label: 'Off', value: 'off' }
  ] },
  { label: 'ocsp_stapling', value: 'https_listen-ocsp_stapling', type: 'bool' },
  { label: '回源协议', value: 'backend_protocol', type: 'select', choices: [
    { label: 'HTTP', value: 'http' },
    { label: 'HTTPS', value: 'https' },
    { label: '跟随协议', value: 'follow' }
  ] },
  { label: '回源HTTP端口', value: 'backend_http_port', type: 'number' },
  { label: '回源HTTPS端口', value: 'backend_https_port', type: 'number' },
  { label: '回源超时', value: 'proxy_timeout', type: 'number' },
  { label: '开启IPv6', value: 'ipv6_enable', type: 'bool' },
  { label: '开启Gzip', value: 'gzip_enable', type: 'bool' },
  { label: '开启Websocket', value: 'websocket_enable', type: 'bool' },
  { label: '上传文件大小限制', value: 'post_size_limit', type: 'number' },
  { label: '数据实时发送', value: 'realtime_send', type: 'bool' },
  { label: '数据实时返回', value: 'realtime_return', type: 'bool' },
  { label: '源站请求头', value: 'origin_headers', type: 'headers' },
  { label: '回源负载方式', value: 'balance_way', type: 'select', choices: [
    { label: '轮询', value: 'rr' },
    { label: '定源', value: 'hash' }
  ] }
]

const currentOption = computed(() => defaultOptions.find(opt => opt.value === form.name))
const currentType = computed(() => currentOption.value?.type || 'text')
const currentChoices = computed(() => {
    if (!currentOption.value) return []
    if (currentOption.value.choicesKey === 'cc_rules') return ccRuleOptions.value
    if (currentOption.value.choicesKey === 'dns_providers') return dnsOptions.value.map(d => ({ label: d.name, value: String(d.id) }))
    return currentOption.value.choices || []
})

const formattedList = computed(() => {
    return list.value.map(item => {
        const option = defaultOptions.find(opt => opt.value === item.name)
        const label = option ? option.label : item.name
        let formattedValue = item.value
        if (option) {
            formattedValue = formatDefaultValue(option.value, item.value, option.type)
        }
        
        let scopeLabel = item.scope_name === 'group' ? '分组' : '全局'
        if (item.scope_name === 'group' && item.group_name) {
            scopeLabel = `分组(${item.group_name})`
        }

        return {
            ...item,
            label,
            value: formattedValue,
            rawValue: item.value,
            scopeLabel
        }
    })
})

const formatDefaultValue = (name, value, type) => {
  if (type === 'bool') {
    return value === '1' || value === 'true' || value === 'on' ? '是' : '否'
  }
  if (type === 'select') {
    const val = value !== undefined && value !== null ? String(value) : ''
    const option = defaultOptions.find(opt => opt.value === name)
    // Need to dynamically resolve choices, but this is a synchronous formatter.
    // For choices that depend on loaded data (like DNS/CC), we might display raw ID if data not loaded
    const choices = (option.choices || [])
    // Note: dynamic choices (cc_rules, dns_providers) might be available in ccRuleOptions/dnsOptions
    if (option.choicesKey === 'cc_rules' && ccRuleOptions.value.length) {
         const match = ccRuleOptions.value.find(c => String(c.value) === val)
         return match ? match.label : val
    }
    if (option.choicesKey === 'dns_providers' && dnsOptions.value.length) {
         const match = dnsOptions.value.find(d => String(d.id) === val)
         return match ? match.name : val
    }

    const match = choices.find(opt => String(opt.value) === val)
    return match ? match.label : val
  }
  if (type === 'headers') {
      try {
          const items = JSON.parse(value || '[]')
          if (Array.isArray(items)) {
              return items.map(i => `${i.name}: ${i.value}`).join('; ')
          }
      } catch(e) {/* ignore */}
  }
  if (type === 'region') {
      if (!value || value === 'none') return '不设置'
      return value // It's a comma separated string of codes usually
  }
  return value
}

// --- Methods ---

const fetchData = async () => {
  loading.value = true
  try {
      const res = await request.get('/site_defaults')
      list.value = res.data?.list || []
  } finally {
      loading.value = false
  }
}

const loadDependencies = async () => {
    // Load groups, dns providers, cc rules for options
    if (props.isAdmin) {
        request.get('/node-groups').then(res => { /* not used directly in default settings form? */ })
    }
    
    // Groups for scope selection
    request.get('/site_groups').then(res => { groupOptions.value = res.data?.list || [] })
    
    // DNS Providers
    request.get('/dnsapi').then(res => { dnsOptions.value = res.list || [] })
    
    // CC Rules
    request.get('/rules/cc/groups', { params: { pageSize: 200 } }).then(res => {
         const list = res.data?.list || res.list || []
         ccRuleOptions.value = [{ label: '不设置', value: '0' }]
             .concat(list.map(item => ({ label: item.name || `规则${item.id}`, value: String(item.id) })))
    })
}

const searchUsers = async (query) => {
    userLoading.value = true
    try {
        const res = await request.get('/users', { params: { keyword: query, pageSize: 20 } })
        userOptions.value = (res.data?.list || []).map(u => ({...u, username: u.username || u.email}))
    } finally {
        userLoading.value = false
    }
}

const handleNameChange = () => {
    // Reset values on type change
    form.value = 0
    form.boolValue = false
    form.selectValue = ''
    form.multiValue = []
    form.textValue = ''
    form.headers = []
    form.region_custom = []
}

const handleAdd = () => {
    mode.value = 'create'
    resetForm()
    visible.value = true
}

const handleEdit = (row) => {
    mode.value = 'edit'
    resetForm()
    
    // Hydrate
    form.name = row.name
    form.scope = row.scope_name === 'group' ? 'group' : 'global'
    form.group_id =  row.scope_name === 'group' ? Number(row.scope_id) : 0
    form.user_id = row.user_id
    
    // Store original ID info for update
    editScope.name = row.name
    editScope.originalName = row.name
    editScope.scopeName = row.scope_name
    editScope.scopeId = row.scope_id

    // Hydrate Value
    hydrateValue(row.name, row.rawValue)
    
    // Load specific user if needed
    if (row.user_id && !userOptions.value.some(u => u.id === row.user_id)) {
        userOptions.value.push({ id: row.user_id, name: row.user_name || String(row.user_id) })
    }
    
    visible.value = true
}

const hydrateValue = (name, value) => {
    const option = defaultOptions.find(o => o.value === name)
    const type = option?.type || 'text'
    
    if (type === 'number') form.value = Number(value) || 0
    else if (type === 'bool') form.boolValue = (value === '1' || value === 'true' || value === 'on')
    else if (type === 'select') form.selectValue = String(value || '')
    else if (type === 'multi') form.multiValue = (value || '').split(/\s+/).filter(Boolean)
    else if (type === 'lines') form.textValue = value || ''
    else if (type === 'headers') {
        try {
            form.headers = JSON.parse(value || '[]')
        } catch(e) { form.headers = [] }
    }
    else if (type === 'region') {
         if (!value || value === 'none') form.region_custom = []
         else form.region_custom = value.split(',').filter(Boolean)
    }
    else form.textValue = value || ''
}

const resetForm = () => {
    form.name = ''
    form.scope = 'global'
    form.group_id = 0
    form.user_id = undefined
    handleNameChange()
}

const addHeader = () => form.headers.push({ name: '', value: '' })
const removeHeader = (idx) => form.headers.splice(idx, 1)

const buildValue = () => {
     if (currentType.value === 'number') return String(form.value)
     if (currentType.value === 'bool') return form.boolValue ? '1' : '0'
     if (currentType.value === 'select') return form.selectValue
     if (currentType.value === 'multi') return form.multiValue.join(' ')
     if (currentType.value === 'lines') return form.textValue
     if (currentType.value === 'headers') return JSON.stringify(form.headers.filter(h => h.name))
     if (currentType.value === 'region') return form.region_custom.length ? form.region_custom.join(',') : 'none'
     return form.textValue
}

const handleSubmit = async () => {
    if (!form.name) return ElMessage.error('请选择设置项')
    if (form.scope === 'group' && !form.group_id) return ElMessage.error('请选择分组')
    if (props.isAdmin && !form.user_id) return ElMessage.error('请选择用户')

    const payload = {
        name: form.name,
        value: buildValue(),
        scope_name: form.scope
    }
    
    if (form.scope === 'group') payload.scope_id = form.group_id
    else payload.scope_id = form.user_id // global scope maps to user_id usually in this system, or 0 maybe? 
    // Wait, List-old says: scopes are 'group' or 'global'. If global, scope_id=user_id. If group, scope_id=group_id.
    
    // Actually, in List-old submitDefault:
    // payload.scope_id = defaultForm.group_id (if group)
    // else payload.scope_id = defaultForm.user_id
    
    if (form.scope !== 'group') payload.scope_id = form.user_id

    if (props.isAdmin) payload.user_id = form.user_id

    try {
        if (mode.value === 'edit') {
            payload.old_scope_name = editScope.scopeName
            payload.old_scope_id = editScope.scopeId
            
            // The API uses name in URL
            await request.put(`/site_defaults/${encodeURIComponent(editScope.originalName)}`, payload)
        } else {
            await request.post('/site_defaults', payload)
        }
        ElMessage.success('保存成功')
        visible.value = false
        fetchData()
    } catch(e) {
        // error handled by interceptor
    }
}

const handleDelete = (row) => {
    ElMessageBox.confirm('确定删除该设置?', '提示').then(async () => {
        const params = { scope_name: row.scope_name, scope_id: row.scope_id }
        if (props.isAdmin && row.user_id) params.user_id = row.user_id
        await request.delete(`/site_defaults/${encodeURIComponent(row.name)}`, { params })
        ElMessage.success('删除成功')
        fetchData()
    })
}

const handleSelectionChange = (rows) => {
    selectedRows.value = rows
}

const handleBatchDelete = async () => {
     ElMessageBox.confirm(`确定删除选中的 ${selectedRows.value.length} 个设置?`, '提示').then(async () => {
        for (const row of selectedRows.value) {
            const params = { scope_name: row.scope_name, scope_id: row.scope_id }
            if (props.isAdmin && row.user_id) params.user_id = row.user_id
            await request.delete(`/site_defaults/${encodeURIComponent(row.name)}`, { params })
        }
        ElMessage.success('批量删除成功')
        selectedRows.value = []
        fetchData()
     })
}

onMounted(() => {
    fetchData()
    loadDependencies()
})
</script>

<style scoped>
.default-settings-container { padding: 10px; }
.toolbar { margin-bottom: 10px; }
.form-tip { font-size: 12px; color: #999; margin-top: 5px; }
.value-text {
    overflow: hidden;
    text-overflow: ellipsis;
    /* white-space: nowrap; - removed for multi-line support with webkit clamp */
}
</style>
