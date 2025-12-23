<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane label="CC规则" name="cc">
        <el-tabs v-model="ccActiveTab">
          <el-tab-pane label="规则组" name="groups">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateGroup">新增分组</el-button>
              <el-select v-model="listQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="禁用" value="off" />
              </el-select>
              <el-input v-model="listQuery.name" placeholder="分组名称，模糊搜索" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchGroups">查询</el-button>
            </div>

            <el-table :data="groupsList" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                <template #default="{row}">{{ row.is_system ? '\u7cfb\u7edf' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统" width="100" align="center">
                <template #default="{row}">
                  <el-tag type="success" v-if="row.is_system" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <el-tag type="success" v-if="row.is_show" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '\u542f\u7528' : '\u7981\u7528' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="sort_order" label="排序" width="80" />
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small" @click="handleEditGroup(row)">编辑</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="匹配器" name="matchers">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateMatcher">新增匹配器</el-button>
              <el-select v-model="matcherListQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="禁用" value="off" />
              </el-select>
              <el-input v-model="matcherListQuery.name" placeholder="匹配器名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchMatchers">查询</el-button>
            </div>

            <el-table :data="matchers" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                <template #default="{row}">{{ row.is_system ? '\u7cfb\u7edf' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统" width="100" align="center">
                <template #default="{row}">
                  <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '\u542f\u7528' : '\u7981\u7528' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small">编辑</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="过滤器" name="filters">
            <div class="filter-container">
              <el-button type="primary" class="filter-item">新增过滤器</el-button>
              <el-select v-model="filterListQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="禁用" value="off" />
              </el-select>
              <el-input v-model="filterListQuery.name" placeholder="过滤器名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchFilters">查询</el-button>
            </div>

            <el-table :data="filters" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                <template #default="{row}">{{ row.is_system ? '\u7cfb\u7edf' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统" width="100" align="center">
                <template #default="{row}">
                  <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column prop="type" label="类型" width="150" />
              <el-table-column label="显示" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '\u542f\u7528' : '\u7981\u7528' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small">编辑</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-tab-pane>

      <el-tab-pane label="ACL规则" name="acl">
        <div class="filter-container">
          <el-button type="primary" class="filter-item" @click="openAclDialog()">新增ACL</el-button>
          <el-select v-model="aclQuery.status" placeholder="状态" class="filter-item" style="width: 120px; margin-left:10px;">
            <el-option label="启用" value="on" />
            <el-option label="禁用" value="off" />
          </el-select>
          <el-input v-model="aclQuery.name" placeholder="ACL名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
          <el-button class="filter-item" type="primary" :icon="Search" @click="fetchAcl">查询</el-button>
        </div>

        <el-table :data="aclList" border fit highlight-current-row style="width: 100%">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="名称" min-width="160" />
          <el-table-column prop="default_action" label="默认动作" width="120">
            <template #default="{row}">
              {{ row.default_action === 'deny' ? '\u62d2\u7edd' : '\u5141\u8bb8' }}
            </template>
          </el-table-column>
          <el-table-column prop="enable" label="用户" width="100">
            <template #default="{row}">
              <el-tag :type="row.enable ? 'success' : 'info'">{{ row.enable ? '\u542f\u7528' : '\u7981\u7528' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="create_time" label="创建时间" width="160" />
          <el-table-column label="操作" width="160" align="center">
            <template #default="{row}">
              <el-button link type="primary" size="small" @click="openAclDialog(row)">编辑</el-button>
              <el-button link type="danger" size="small" @click="deleteAcl(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="800px">
      <el-form :model="tempGroup" label-position="right" label-width="100px" style="width: 700px; margin-left:50px;">
        <el-form-item label="类型">
          <el-radio-group v-model="tempGroup.type">
            <el-radio label="system">系统</el-radio>
            <el-radio label="user">用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="tempGroup.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="tempGroup.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item label="规则">
          <el-button type="primary" plain size="small" @click="handleAddRule">新增规则</el-button>
          <el-table :data="tempGroup.rules" border style="width: 100%; margin-top: 10px;" size="small">
            <el-table-column prop="matcher_name" label="匹配器" />
            <el-table-column prop="filter1_name" label="过滤器" />
            <el-table-column prop="action" label="允许" />
            <el-table-column label="拒绝" width="60">
              <template #default="{$index}">
                <el-button link type="danger" @click="removeRule($index)">取消</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">确定</el-button>
          <el-button type="primary" @click="saveGroup">新增</el-button>
        </div>
      </template>

      <el-dialog width="600px" v-model="innerVisible" title="新增规则" append-to-body>
        <el-form :model="tempRule" label-width="100px">
          <el-form-item label="匹配器">
            <el-select v-model="tempRule.matcher_id" placeholder="请选择匹配器" style="width: 100%">
              <el-option v-for="item in matchers" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="过滤器">
            <el-select v-model="tempRule.filter1_id" placeholder="请选择匹配器" style="width: 100%">
              <el-option v-for="item in filters" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="动作">
            <el-select v-model="tempRule.action" placeholder="请选择动作" style="width: 100%">
              <el-option label="阻断" value="block" />
              <el-option label="放行" value="allow" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <div class="dialog-footer">
            <el-button @click="innerVisible = false">删除</el-button>
            <el-button type="primary" @click="confirmAddRule">取消</el-button>
          </div>
        </template>
      </el-dialog>
    </el-dialog>

    <el-dialog title="新增匹配器" v-model="matcherDialogVisible" width="800px">
      <el-form :model="tempMatcher" label-width="80px">
        <el-form-item label="类型">
          <el-radio-group v-model="tempMatcher.type">
            <el-radio label="system">系统</el-radio>
            <el-radio label="user">用户</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="tempMatcher.name" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="tempMatcher.remark" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="规则">
          <div style="width: 100%">
            <el-row :gutter="10" style="margin-bottom: 5px; font-weight: bold; color: #606266; font-size:12px;">
              <el-col :span="6">条件项</el-col>
              <el-col :span="4">操作符</el-col>
              <el-col :span="10">匹配值</el-col>
              <el-col :span="4">操作</el-col>
            </el-row>
            <div v-for="(rule, index) in tempMatcher.rules" :key="index" style="margin-bottom: 10px;">
              <el-row :gutter="10">
                <el-col :span="6">
                  <el-select v-model="rule.item" placeholder="请选择" size="small">
                    <el-option label="IP地址" value="ip" />
                    <el-option label="URL" value="url" />
                    <el-option label="User-Agent" value="ua" />
                    <el-option label="Referer" value="referer" />
                  </el-select>
                </el-col>
                <el-col :span="4">
                  <el-select v-model="rule.operator" placeholder="请选择" size="small">
                    <el-option label="等于" value="eq" />
                    <el-option label="包含" value="contains" />
                    <el-option label="正则匹配" value="regex" />
                  </el-select>
                </el-col>
                <el-col :span="10">
                  <el-input v-model="rule.value" placeholder="请输入匹配值" type="textarea" :rows="1" size="small" />
                </el-col>
                <el-col :span="4">
                  <el-button type="primary" link @click="addMatcherRule">确定</el-button>
                  <el-button type="danger" link @click="removeMatcherRule(index)" v-if="tempMatcher.rules.length > 1">删除</el-button>
                </el-col>
              </el-row>
            </div>
          </div>
          <div style="font-size: 12px; color: #999; margin-top: 5px;">
            同一规则内条件为 AND，不同规则之间为 OR
          </div>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="tempMatcher.is_on" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="matcherDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveMatcher">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="aclDialogVisible" :title="aclDialogTitle" width="680px">
      <el-form :model="aclForm" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="aclForm.name" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="aclForm.des" placeholder="请选择动作" />
        </el-form-item>
        <el-form-item label="默认动作">
          <el-radio-group v-model="aclForm.default_action">
            <el-radio label="allow">允许</el-radio>
            <el-radio label="deny">拒绝</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="aclForm.enable" />
        </el-form-item>
        <el-form-item label="规则">
          <el-button type="primary" plain size="small" @click="addAclRule">新增规则</el-button>
          <el-table :data="aclForm.rules" border style="width: 100%; margin-top: 10px;" size="small">
            <el-table-column label="IP">
              <template #default="{ row }">
                <el-input v-model="row.ip" placeholder="IP?CIDR" />
              </template>
            </el-table-column>
            <el-table-column label="动作" width="140">
              <template #default="{ row }">
                <el-select v-model="row.action" style="width: 100%;">
                  <el-option label="放行" value="allow" />
                  <el-option label="拒绝" value="deny" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="80">
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeAclRule($index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="aclDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAcl">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { Search, ArrowDown, Select } from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import request from '@/utils/request'

const activeTab = ref('cc')
const ccActiveTab = ref('groups')
const groupsList = ref([])
const matchers = ref([])
const filters = ref([])
const aclList = ref([])

const listQuery = reactive({
  name: '',
  status: ''
})
const matcherListQuery = reactive({
  name: '',
  status: ''
})
const filterListQuery = reactive({
  name: '',
  status: ''
})
const aclQuery = reactive({
  name: '',
  status: ''
})

const dialogFormVisible = ref(false)
const innerVisible = ref(false)
const matcherDialogVisible = ref(false)
const dialogStatus = ref('')
const textMap = {
  update: '\u7f16\u8f91\u89c4\u5219\u7ec4',
  create: '\u65b0\u589e\u89c4\u5219\u7ec4'
}

const tempGroup = reactive({
  id: undefined,
  type: 'system',
  name: '',
  remark: '',
  rules: []
})

const tempRule = reactive({
  matcher_id: undefined,
  matcher_name: '',
  filter1_id: undefined,
  filter1_name: '',
  action: 'block'
})

const tempMatcher = reactive({
  type: 'system',
  name: '',
  remark: '',
  is_on: true,
  rules: [
    { item: 'ip', operator: 'eq', value: '' }
  ]
})

const aclDialogVisible = ref(false)
const aclForm = reactive({
  id: 0,
  name: '',
  des: '',
  default_action: 'allow',
  enable: true,
  rules: []
})
const aclDialogTitle = computed(() => (aclForm.id ? '\u7f16\u8f91ACL' : '\u65b0\u589eACL'))

const fetchGroups = async () => {
  const { data } = await request.get('/rules/cc/groups', { params: listQuery })
  groupsList.value = data.list || []
}

const fetchMatchers = async () => {
  const { data } = await request.get('/rules/cc/matchers', { params: matcherListQuery })
  matchers.value = data.list || []
}

const fetchFilters = async () => {
  const { data } = await request.get('/rules/cc/filters', { params: filterListQuery })
  filters.value = data.list || []
}

const fetchAcl = async () => {
  const { data } = await request.get('/rules/acl', { params: aclQuery })
  aclList.value = data.list || []
}

const handleCreateGroup = () => {
  dialogStatus.value = 'create'
  dialogFormVisible.value = true
  Object.assign(tempGroup, { id: undefined, type: 'system', name: '', remark: '', rules: [] })
}

const handleEditGroup = async row => {
  dialogStatus.value = 'update'
  dialogFormVisible.value = true
  const { data } = await request.get(`/rules/cc/groups/${row.id}`)
  Object.assign(tempGroup, data)
}

const saveGroup = () => {
  dialogFormVisible.value = false
  fetchGroups()
}

const handleAddRule = async () => {
  if (matchers.value.length === 0) await fetchMatchers()
  if (filters.value.length === 0) await fetchFilters()
  Object.assign(tempRule, {
    matcher_id: undefined,
    matcher_name: '',
    filter1_id: undefined,
    filter1_name: '',
    action: 'block'
  })
  innerVisible.value = true
}

const confirmAddRule = () => {
  const m = matchers.value.find(i => i.id === tempRule.matcher_id)
  if (m) tempRule.matcher_name = m.name
  const f1 = filters.value.find(i => i.id === tempRule.filter1_id)
  if (f1) tempRule.filter1_name = f1.name
  tempGroup.rules.push({ ...tempRule })
  innerVisible.value = false
}

const removeRule = index => {
  tempGroup.rules.splice(index, 1)
}

const handleCreateMatcher = () => {
  matcherDialogVisible.value = true
  Object.assign(tempMatcher, {
    type: 'system',
    name: '',
    remark: '',
    is_on: true,
    rules: [{ item: 'ip', operator: 'eq', value: '' }]
  })
}

const addMatcherRule = () => {
  tempMatcher.rules.push({ item: 'ip', operator: 'eq', value: '' })
}

const removeMatcherRule = index => {
  tempMatcher.rules.splice(index, 1)
}

const saveMatcher = () => {
  matcherDialogVisible.value = false
  fetchMatchers()
}

const openAclDialog = async row => {
  if (row && row.id) {
    const { data } = await request.get(`/rules/acl/${row.id}`)
    aclForm.id = data.id
    aclForm.name = data.name
    aclForm.des = data.des || ''
    aclForm.default_action = data.default_action || 'allow'
    aclForm.enable = !!data.enable
    aclForm.rules = data.rules || []
  } else {
    aclForm.id = 0
    aclForm.name = ''
    aclForm.des = ''
    aclForm.default_action = 'allow'
    aclForm.enable = true
    aclForm.rules = []
  }
  aclDialogVisible.value = true
}

const saveAcl = async () => {
  const payload = {
    name: aclForm.name,
    des: aclForm.des,
    default_action: aclForm.default_action,
    enable: aclForm.enable,
    rules: aclForm.rules
  }
  if (aclForm.id) {
    await request.put(`/rules/acl/${aclForm.id}`, payload)
  } else {
    await request.post('/rules/acl', payload)
  }
  ElMessage.success('\u4fdd\u5b58\u6210\u529f')
  aclDialogVisible.value = false
  fetchAcl()
}

const deleteAcl = row => {
  ElMessageBox.confirm('\u786e\u5b9a\u5220\u9664ACL\uff1f', '\u63d0\u793a', {
    confirmButtonText: '\u786e\u5b9a',
    cancelButtonText: '\u53d6\u6d88',
    type: 'warning'
  }).then(() => {
    request.delete(`/rules/acl/${row.id}`).then(() => {
      ElMessage.success('\u4fdd\u5b58\u6210\u529f')
      fetchAcl()
    })
  })
}

const addAclRule = () => {
  aclForm.rules.push({ ip: '', action: 'allow' })
}

const removeAclRule = index => {
  aclForm.rules.splice(index, 1)
}

onMounted(() => {
  fetchGroups()
  fetchMatchers()
  fetchFilters()
  fetchAcl()
})
</script>

<style scoped>
.filter-container {
  padding-bottom: 20px;
}
</style>
