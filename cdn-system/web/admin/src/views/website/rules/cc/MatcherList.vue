<template>
  <div class="filter-container">
    <el-button type="primary" size="small" @click="handleCreate">新增匹配器</el-button>
    <el-input v-model="query.name" placeholder="名称" style="width: 150px; margin-left: 10px;" size="small" />
    <el-button size="small" type="primary" :icon="Search" @click="fetchData">查询</el-button>
  </div>

  <AppTable :data="list" :loading="loading" border fit highlight-current-row persist-key="cc-matchers">
    <el-table-column type="selection" width="55" />
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column v-if="isAdmin" prop="user" label="用户" width="120">
      <template #default="{row}">
        <span v-if="row.is_system">系统</span>
        <span v-else>{{ row.user?.username || row.user_id }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="name" label="名称" min-width="150" />
    <el-table-column label="类型" width="100" align="center">
      <template #default="{row}">
        <span>{{ row.type_label || (row.is_system ? '系统规则' : '用户规则') }}</span>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="80" align="center">
      <template #default="{row}">
        <span :class="row.is_on ? 'text-success' : 'text-danger'">{{ row.is_on ? '正常' : '禁用' }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="created_at" label="创建时间" width="160">
      <template #default="{row}">{{ formatTime(row.created_at) }}</template>
    </el-table-column>
    <el-table-column label="操作" width="140" align="center">
      <template #default="{row}">
        <div class="action-cell">
          <el-button type="primary" link size="small" @click="handleEdit(row)">管理</el-button>
          <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, row)">
            <el-button type="primary" link size="small">更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="!row.is_system" command="delete" style="color: #F56C6C;">删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </template>
    </el-table-column>
  </AppTable>

  <el-dialog :title="dialogMode === 'create' ? '新增匹配器' : (isSystemRuleEdit ? '查看匹配器（系统规则只读）' : '编辑匹配器')" v-model="dialogVisible" width="800px" :close-on-click-modal="false">
    <el-alert v-if="isSystemRuleEdit" type="info" :closable="false" show-icon title="系统规则为只读，管理员也不可修改" style="margin-bottom: 12px;" />
    <el-form :model="form" label-width="100px" :disabled="isSystemRuleEdit">
      <el-form-item label="类型" v-if="isAdmin">
        <el-radio-group v-model="form.type" :disabled="isSystemRuleEdit">
          <el-radio value="system">系统规则</el-radio>
          <el-radio value="user">用户规则</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="用户" v-if="isAdmin && form.type === 'user'">
        <el-select
          v-model="form.user_id"
          filterable
          remote
          placeholder="请输入ID或账号搜索"
          :remote-method="searchUsers"
          :loading="userLoading"
          style="width: 100%"
        >
          <el-option v-for="item in userOptions" :key="item.id" :label="item.username" :value="item.id">
            <span style="float: left">{{ item.username }}</span>
            <span style="float: right; color: #8492a6; font-size: 13px">ID:{{ item.id }}</span>
          </el-option>
        </el-select>
      </el-form-item>
       <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
       <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>
       
       <el-form-item label="规则列表">
          <el-table :data="form.rules" border size="small">
             <el-table-column label="匹配项">
                <template #default="{row}">{{ getMatchItemLabel(row.item) }}</template>
             </el-table-column>
             <el-table-column label="操作符" width="120">
                <template #default="{row}">{{ getOperatorLabel(row.operator) }}</template>
             </el-table-column>
             <el-table-column label="匹配值">
                <template #default="{row}">{{ row.value }}</template>
             </el-table-column>
             <el-table-column label="操作" width="130" align="center">
                <template #default="{$index}">
                   <el-button link type="primary" size="small" @click="moveRule($index, -1)" :disabled="$index === 0">
                     <el-icon><ArrowUp /></el-icon>
                   </el-button>
                   <el-button link type="primary" size="small" @click="moveRule($index, 1)" :disabled="$index === form.rules.length - 1">
                     <el-icon><ArrowDown /></el-icon>
                   </el-button>
                   <el-button link type="danger" @click="removeRow($index)">移除</el-button>
                </template>
             </el-table-column>
          </el-table>
          
          <div class="add-rule-box">
            <el-select v-model="newRule.item" size="small" placeholder="匹配项" style="width: 180px">
                <el-option v-for="opt in matchOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <el-select v-model="newRule.operator" size="small" placeholder="操作符" style="width: 120px">
                <el-option v-for="opt in operatorOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <el-input v-model="newRule.value" size="small" placeholder="输入匹配值(可为空)" style="flex: 1;" />
            <el-button type="primary" link size="small" @click="addRow">添加</el-button>
          </div>
          <div class="form-helper">规则从上往下匹配，可使用按钮来调整顺序。</div>
          <div class="form-helper" v-if="newRule.item === 'header'">多个匹配条件的关系为且，即所有条件都满足时才执行下面的过滤</div>
       </el-form-item>
       
      <el-form-item label="启用">
        <el-switch v-model="form.is_on" :disabled="isSystemRuleEdit || cannotDisable" />
        <div v-if="cannotDisable" class="form-helper">该匹配器正在使用中，无法禁用</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">{{ isSystemRuleEdit ? '关闭' : '取消' }}</el-button>
      <el-button v-if="!isSystemRuleEdit" type="primary" @click="submitForm">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { Search, Check, Close, ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const loading = ref(false)
const query = reactive({ name: '' })
const dialogVisible = ref(false)
const dialogMode = ref('create')
const isAdmin = ref(localStorage.getItem('role') === 'admin')
const form = reactive({ id: 0, type: 'system', user_id: null, name: '', remark: '', rules: [], is_on: true, is_system: false, in_use: false })

const isSystemRuleEdit = computed(() => dialogMode.value === 'update' && (form.type === 'system' || form.is_system))
const cannotDisable = computed(() => !!form.in_use && !!form.is_on)

const userLoading = ref(false)
const userOptions = ref([])

const newRule = reactive({ item: 'all', operator: 'eq', value: '' })

const resolveRuleType = (item = {}) => {
  if (item.type === 'system' || item.type === 'user') return item.type
  return item.is_system ? 'system' : 'user'
}

const resolveUserId = (...items) => {
  for (const item of items) {
    const userId = item?.user_id ?? item?.uid
    if (userId) return userId
  }
  return null
}

const matchOptions = [
  { label: '匹配所有请求', value: 'all' },
  { label: 'IP地址', value: 'ip' },
  { label: '域名', value: 'domain' },
  { label: '请求URI', value: 'request_uri' },
  { label: '请求URI(不带参数)', value: 'request_path' },
  { label: '请求头', value: 'header' },
  { label: '独立UA数量', value: 'user_agent_unique_count' },
  { label: '404状态码数量', value: 'status_404_count' },
  { label: '请求方法', value: 'method' },
  { label: '浏览器UA', value: 'user_agent' },
  { label: '请求来源', value: 'referer' },
  { label: '国家代码', value: 'country' },
  { label: 'AS号码', value: 'asn' },
  { label: '省份', value: 'province' },
  { label: '城市', value: 'city' },
  { label: '运营商', value: 'isp' },
  { label: 'HTTP版本', value: 'http_version' },
  { label: '请求头accept_language', value: 'accept_language' }
]

const operatorOptions = [
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'neq' },
  { label: '包含', value: 'contains' },
  { label: '不包含', value: 'not_contains' },
  { label: '前缀匹配', value: 'prefix' },
  { label: '后缀匹配', value: 'suffix' },
  { label: '正则匹配', value: 'regex' },
  { label: '正则不匹配', value: 'not_regex' },
  { label: '存在', value: 'exists' },
  { label: '不存在', value: 'not_exists' },
  { label: '在IP段', value: 'ip_range' },
  { label: '不在IP段', value: 'not_ip_range' }
]

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await request.get('/rules/cc/matchers', { params: query })
    list.value = data.list || []
  } finally {
    loading.value = false
  }
}

const searchUsers = async (query) => {
  if (!isAdmin.value) return
  userLoading.value = true
  try {
    const { data } = await request.get('/users', { params: { keyword: query, size: 20 } })
    userOptions.value = data.list || []
  } finally {
    userLoading.value = false
  }
}

const handleCreate = () => {
  dialogMode.value = 'create'
  Object.assign(form, { 
    id: 0, 
    type: isAdmin.value ? 'system' : 'user', 
    user_id: null, 
    name: '', 
    remark: '', 
    rules: [], 
    is_on: true 
  })
  // Reset new rule input
  Object.assign(newRule, { item: 'all', operator: 'eq', value: '' })
  dialogVisible.value = true
}

const handleEdit = async (row) => {
  dialogMode.value = 'update'
  const { data } = await request.get(`/rules/cc/matchers/${row.id}`)
  Object.assign(form, data)
  form.type = resolveRuleType(data)
  form.is_system = !!data.is_system
  form.in_use = !!data.in_use
  form.user_id = form.type === 'user' ? resolveUserId(data, row) : null
  if (!form.rules) form.rules = []

  // Load specific user for label
  if (form.user_id && isAdmin.value) {
    try {
      const { data: user } = await request.get(`/users/${form.user_id}`)
      if (user) {
        userOptions.value = [user]
      }
    } catch(e) { /* ignore if user not found */ }
  }
  
  dialogVisible.value = true
}

const addRow = () => {
  form.rules.push({ ...newRule })
  // Reset value only maybe? Or keep for convenience? Usually clear.
  newRule.value = ''
}

const removeRow = (idx) => form.rules.splice(idx, 1)

const moveRule = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= form.rules.length) return
  const temp = form.rules[index]
  form.rules[index] = form.rules[newIndex]
  form.rules[newIndex] = temp
}

const submitForm = async () => {
    if (isSystemRuleEdit.value) {
      dialogVisible.value = false
      return
    }
    try {
      if (isAdmin.value && form.type === 'user' && !form.user_id) {
        ElMessage.warning('请选择用户')
        return
      }
      const payload = { ...form, user_id: form.type === 'system' ? null : form.user_id }
      if (form.id) await request.put(`/rules/cc/matchers/${form.id}`, payload)
      else await request.post('/rules/cc/matchers', payload)
      ElMessage.success('保存成功')
      dialogVisible.value = false
      fetchData()
    } catch(e) {}
}

watch(() => form.type, (type) => {
  if (type === 'system') {
    form.user_id = null
  }
})

const handleCommand = (cmd, row) => {
  if (cmd === 'delete') handleDelete(row)
}

const handleDelete = (row) => {
  if (row.is_system) {
    ElMessage.warning('系统规则不可删除')
    return
  }
  ElMessageBox.confirm('确定删除匹配器?', '提示', { type: 'warning' }).then(async () => {
    await request.delete(`/rules/cc/matchers/${row.id}`)
    ElMessage.success('删除成功')
    fetchData()
  })
}

const getMatchItemLabel = (val) => matchOptions.find(o => o.value === val)?.label || val
const getOperatorLabel = (val) => operatorOptions.find(o => o.value === val)?.label || val

const formatTime = (ts) => {
  if (!ts) return '-'
  const date = new Date(ts * 1000)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${m}-${d} ${h}:${min}:${s}`
}

onMounted(() => {
  fetchData()
  if (isAdmin.value) searchUsers('')
})
</script>

<style scoped>
.filter-container {
  padding-bottom: 10px;
  display: flex;
  align-items: center;
}
.text-success { color: #67C23A; }
.text-danger { color: #F56C6C; }
.action-cell {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
}
.form-helper {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  margin-top: 5px;
}
.add-rule-box {
  display: flex;
  gap: 10px;
  margin-top: 10px;
  align-items: center;
  border: 1px dashed #dcdfe6;
  padding: 10px;
  border-radius: 4px;
}
</style>
