<template>
  <div class="filter-container">
    <el-button type="primary" size="small" @click="handleCreate">新增分组</el-button>
    <el-select v-model="query.status" placeholder="状态" size="small" style="width: 100px; margin-left:10px;">
      <el-option label="启用" value="on" />
      <el-option label="禁用" value="off" />
    </el-select>
    <el-input v-model="query.name" placeholder="名称" style="width: 150px; margin-left: 10px;" size="small" />
    <el-button size="small" type="primary" :icon="Search" @click="fetchData">查询</el-button>
  </div>

  <AppTable :data="list" :loading="loading" border fit highlight-current-row persist-key="cc-groups">
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
    <el-table-column label="显示" width="80" align="center">
      <template #default="{row}">
        <el-switch v-model="row.is_visible" disabled size="small" />
      </template>
    </el-table-column>
    <el-table-column label="状态" width="80" align="center">
      <template #default="{row}">
        <span :class="row.is_on ? 'text-success' : 'text-danger'">{{ row.is_on ? '正常' : '禁用' }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="sort_order" label="排序" width="80" />
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

  <!-- Group Form Dialog -->
  <el-dialog :title="dialogMode === 'create' ? '新增规则组' : '编辑规则组'" v-model="dialogVisible" width="800px" :close-on-click-modal="false">
    <el-form :model="form" label-width="100px">
      <el-form-item label="类型" v-if="isAdmin">
        <el-radio-group v-model="form.type" :disabled="isSystemRule">
          <el-radio value="system">系统规则</el-radio>
          <el-radio value="user">用户规则</el-radio>
        </el-radio-group>
        <div v-if="isSystemRule" class="form-helper">系统规则的类型不可修改，其他配置可正常编辑</div>
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
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="请输入规则组名称" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" placeholder="请输入备注" />
      </el-form-item>

      <el-form-item label="启用">
        <el-switch v-model="form.is_on" :disabled="cannotDisable" />
        <div v-if="cannotDisable" class="form-helper">该规则组正在使用中，无法禁用</div>
      </el-form-item>
      
      <el-form-item label="规则列表">
        <div style="width: 100%">
          <el-button type="primary" plain size="small" @click="openRuleDialog" style="margin-bottom: 8px;">新增规则</el-button>
          <el-table :data="form.rules" border size="small">
            <el-table-column label="匹配器" min-width="100">
              <template #default="{row}">{{ getMatcherName(row.matcher_id) }}</template>
            </el-table-column>
            <el-table-column label="过滤器1" min-width="100">
               <template #default="{row}">{{ getFilterName(row.filter1_id) }}</template>
            </el-table-column>
            <el-table-column label="过滤器2" min-width="100">
               <template #default="{row}">{{ getFilterName(row.filter2_id) }}</template>
            </el-table-column>
            <el-table-column prop="action" label="动作" width="80">
              <template #default="{row}">{{ transformAction(row.action) }}</template>
            </el-table-column>
             <el-table-column label="模式" width="80">
               <template #default="{row}">{{ row.mode === 'stop' ? '停止' : '继续' }}</template>
            </el-table-column>
            <el-table-column label="启用" width="70" align="center">
              <template #default="{row}">
                 <el-switch v-model="row.is_on" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="130" align="center">
              <template #default="{$index}">
                <el-button link type="primary" size="small" @click="moveRule($index, -1)" :disabled="$index === 0">
                  <el-icon><ArrowUp /></el-icon>
                </el-button>
                <el-button link type="primary" size="small" @click="moveRule($index, 1)" :disabled="$index === form.rules.length - 1">
                  <el-icon><ArrowDown /></el-icon>
                </el-button>
                <el-button link type="danger" size="small" @click="removeRule($index)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="form-helper">规则从上往下匹配，可使用按钮来调整顺序。</div>
        </div>
      </el-form-item>

      <div class="more-settings-container">
        <div class="section-trigger" @click="showMoreSettings = !showMoreSettings">
          <span>更多设置</span>
          <el-icon><component :is="showMoreSettings ? 'ArrowUp' : 'ArrowDown'" /></el-icon>
        </div>
        <div v-show="showMoreSettings" class="more-settings-content">
          <el-form-item label="是否显示">
            <el-switch v-model="form.is_visible" />
          </el-form-item>
          <el-form-item label="指定显示" v-if="form.type === 'system'">
             <el-select
              v-model="form.visible_users"
              multiple
              filterable
              remote
              placeholder="请输入ID或账号搜索"
              :remote-method="searchUsers"
              :loading="userLoading"
              style="width: 100%"
            >
              <el-option v-for="item in userOptions" :key="item.id" :label="item.username" :value="item.id" />
            </el-select>
             <div class="form-helper">指定用户可见，留空则对所有用户可见（仅限系统规则）</div>
          </el-form-item>
           <el-form-item label="排序">
            <el-input-number v-model="form.sort_order" :min="0" controls-position="right" />
          </el-form-item>
        </div>
      </div>

    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitForm">确定</el-button>
    </template>

    <!-- Inner Rule Dialog -->
    <el-dialog v-model="ruleDialogVisible" title="规则设置" width="550px" append-to-body :close-on-click-modal="false">
      <el-form :model="ruleForm" label-width="100px">
        <el-form-item label="匹配器" required>
          <el-select v-model="ruleForm.matcher_id" style="width: 100%" placeholder="选择匹配器">
            <el-option v-for="m in filteredMatchers" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
           <div class="form-helper" v-if="form.type === 'system'">仅显示系统匹配器</div>
        </el-form-item>
        <el-form-item label="过滤器1">
          <el-select v-model="ruleForm.filter1_id" style="width: 100%" placeholder="选择过滤器" clearable>
            <el-option v-for="f in filteredFilters" :key="f.id" :label="f.name" :value="f.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="过滤器2">
          <el-select v-model="ruleForm.filter2_id" style="width: 100%" placeholder="选择过滤器" clearable>
            <el-option v-for="f in filteredFilters" :key="f.id" :label="f.name" :value="f.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="动作">
          <el-radio-group v-model="ruleForm.action">
            <el-radio value="block">拉黑</el-radio>
            <el-radio value="log">只记录</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="匹配模式">
          <el-radio-group v-model="ruleForm.mode">
            <el-radio value="continue">继续下一条规则</el-radio>
            <el-radio value="stop">停止匹配</el-radio>
          </el-radio-group>
          <div class="form-helper">
            1. 模式为继续下一条规则时，执行当前过滤器后，继续下一条规则匹配；<br>
            2. 模式为停止匹配时，执行当前过滤器后，不再继续下一条规则匹配。
          </div>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.is_on" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="addRule">确定</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { Search, Check, Close, ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const loading = ref(false)
const query = reactive({ name: '', status: '' })
const dialogVisible = ref(false)
const dialogMode = ref('create')
const isAdmin = ref(localStorage.getItem('role') === 'admin')
const showMoreSettings = ref(false)

const form = reactive({ 
  id: 0, 
  type: 'system', 
  name: '', 
  remark: '', 
  user_id: null,
  is_visible: true,
  visible_users: [],
  sort_order: 0,
  is_on: true,
  in_use: false,
  rules: [] 
})

const isSystemRule = computed(() => dialogMode.value === 'update' && (form.type === 'system' || form.is_system))
const cannotDisable = computed(() => !!form.in_use && !!form.is_on)

const matchers = ref([])
const filters = ref([])
const ruleDialogVisible = ref(false)
const ruleForm = reactive({ matcher_id: null, filter1_id: null, filter2_id: null, action: 'block', mode: 'continue', is_on: true })

const userLoading = ref(false)
const userOptions = ref([])

const filteredMatchers = computed(() => {
  if (form.type === 'system') {
    return matchers.value.filter(m => m.is_system)
  } else {
    return matchers.value.filter(m => !m.is_system)
  }
})

const filteredFilters = computed(() => {
  if (form.type === 'system') {
    return filters.value.filter(f => f.is_system)
  } else {
    return filters.value.filter(f => !f.is_system)
  }
})

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await request.get('/rules/cc/groups', { params: query })
    list.value = data.list || []
  } catch (e) {
    // ignore
  } finally {
    loading.value = false
  }
}

const loadMatchers = async () => {
  const { data } = await request.get('/rules/cc/matchers')
  matchers.value = data.list || [] // Ensure list has is_public field
}

const loadFilters = async () => {
  const { data } = await request.get('/rules/cc/filters')
  filters.value = data.list || []
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
    name: '', 
    remark: '', 
    rules: [],
    user_id: null,
    is_visible: true,
    visible_users: [],
    sort_order: 0,
    is_on: true,
    in_use: false
  })
  showMoreSettings.value = false
  dialogVisible.value = true
}

const handleEdit = async (row) => {
  dialogMode.value = 'update'
  try {
    const { data } = await request.get(`/rules/cc/groups/${row.id}`)
    Object.assign(form, data)
    form.type = data.type || (data.is_system ? 'system' : 'user')
    form.is_system = !!data.is_system
    form.in_use = !!data.in_use
    if (!form.rules) form.rules = []
    if (!form.visible_users) form.visible_users = []
    
    showMoreSettings.value = false
    dialogVisible.value = true
    
    // Pre-load user if updating and user_id is set
    if (form.user_id) {
       // Ideally fetch specific user info to populate select placeholder/option
       // For now simulate by searching empty string or keep it simple
       searchUsers('')
    }
  } catch(e){}
}

const submitForm = async () => {
  try {
    if (form.id) {
      await request.put(`/rules/cc/groups/${form.id}`, form)
    } else {
      await request.post('/rules/cc/groups', form)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch (err) {}
}

const handleCommand = (cmd, row) => {
  if (cmd === 'delete') {
    handleDelete(row)
  }
}

const handleDelete = (row) => {
  if (row.is_system) {
    ElMessage.warning('系统规则不可删除')
    return
  }
  ElMessageBox.confirm('确定删除规则组?', '提示', { type: 'warning' }).then(async () => {
    await request.delete(`/rules/cc/groups/${row.id}`)
    ElMessage.success('删除成功')
    fetchData()
  })
}

const openRuleDialog = async () => {
  if (!matchers.value.length) loadMatchers()
  if (!filters.value.length) loadFilters()
  
  Object.assign(ruleForm, { matcher_id: null, filter1_id: null, filter2_id: null, action: 'block', mode: 'continue', is_on: true })
  ruleDialogVisible.value = true
}

const addRule = () => {
  if (!ruleForm.matcher_id) {
    ElMessage.warning('请选择匹配器')
    return
  }
  form.rules.push({ ...ruleForm })
  ruleDialogVisible.value = false
}

const removeRule = (idx) => form.rules.splice(idx, 1)

const moveRule = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= form.rules.length) return
  const temp = form.rules[index]
  form.rules[index] = form.rules[newIndex]
  form.rules[newIndex] = temp
}

const getMatcherName = (id) => matchers.value.find(m => m.id === id)?.name || id
const getFilterName = (id) => filters.value.find(f => f.id === id)?.name || id
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
const transformAction = (act) => {
  const map = { block: '拉黑', allow: '放行', log: '只记录', captcha: '验证码' } // Extended map
  return map[act] || act
}

onMounted(() => {
  fetchData()
  if (isAdmin.value) {
    searchUsers('') // Preload some users?
  }
  // Preload matchers/filters for list display names?
  loadMatchers()
  loadFilters()
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
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  margin-top: 5px;
}
.more-settings-container {
  border-top: 1px dashed var(--el-border-color-lighter);
  margin-top: 10px;
  padding-top: 10px;
}
.section-trigger {
  cursor: pointer;
  color: #409EFF;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 10px;
  user-select: none;
}
.more-settings-content {
  padding-left: 20px;
}
</style>
