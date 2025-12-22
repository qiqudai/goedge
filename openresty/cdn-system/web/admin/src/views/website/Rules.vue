<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane label="CC规则" name="cc">
        <el-tabs v-model="ccActiveTab">
          <el-tab-pane label="规则组" name="groups">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateGroup">添加规则组</el-button>
              <el-select v-model="listQuery.status" placeholder="更多操作" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="停用" value="off" />
              </el-select>
              <el-input v-model="listQuery.name" placeholder="规则组名称, 模糊搜索" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" icon="Search" @click="fetchGroups">搜索</el-button>
            </div>

            <el-table :data="groupsList" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                <template #default="{row}">{{ row.is_system ? '系统' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统规则" width="100" align="center">
                <template #default="{row}">
                   <el-tag type="success" v-if="row.is_system" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="显示" width="100" align="center">
                 <template #default="{row}">
                   <el-tag type="success" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="100" align="center">
                <template #default="{row}">
                  <span style="color: #67C23A">● 正常</span>
                </template>
              </el-table-column>
              <el-table-column prop="sort_order" label="排序" width="80" />
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small" @click="handleEditGroup(row)">管理</el-button>
                  <el-button type="primary" link size="small">更多 <el-icon><ArrowDown /></el-icon></el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="匹配器" name="matchers">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateMatcher">添加匹配器</el-button>
              <el-select v-model="matcherListQuery.status" placeholder="更多操作" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="停用" value="off" />
              </el-select>
              <el-input v-model="matcherListQuery.name" placeholder="名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" icon="Search" @click="fetchMatchers">搜索</el-button>
            </div>

            <el-table :data="matchers" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                 <template #default="{row}">{{ row.is_system ? '系统' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统规则" width="100" align="center">
                <template #default="{row}">
                   <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="100" align="center">
                <template #default="{row}">
                  <span style="color: #67C23A">● {{ row.status }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small">管理</el-button>
                  <el-button type="primary" link size="small">更多 <el-icon><ArrowDown /></el-icon></el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="过滤器" name="filters">
            <div class="filter-container">
              <el-button type="primary" class="filter-item">添加过滤器</el-button>
              <el-select v-model="filterListQuery.status" placeholder="更多操作" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="启用" value="on" />
                <el-option label="停用" value="off" />
              </el-select>
              <el-input v-model="filterListQuery.name" placeholder="名称" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" icon="Search" @click="fetchFilters">搜索</el-button>
            </div>

            <el-table :data="filters" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="用户" width="100">
                 <template #default="{row}">{{ row.is_system ? '系统' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="名称" />
              <el-table-column label="系统规则" width="100" align="center">
                 <template #default="{row}">
                   <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column prop="type" label="类型" width="150" />
              <el-table-column label="状态" width="100" align="center">
                <template #default="{row}">
                  <span style="color: #67C23A">● {{ row.status }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="创建时间" width="160" />
              <el-table-column label="操作" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small">管理</el-button>
                  <el-button type="primary" link size="small">更多 <el-icon><ArrowDown /></el-icon></el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-tab-pane>
      <el-tab-pane label="ACL管理" name="acl">ACL管理内容 (待开发)</el-tab-pane>
    </el-tabs>

    <!-- Dialog: Group Edit -->
    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="800px">
      <!-- ... (Group Edit Content stays same) ... -->
      <el-form :model="tempGroup" label-position="right" label-width="100px" style="width: 700px; margin-left:50px;">
        <el-form-item label="类型:">
          <el-radio-group v-model="tempGroup.type">
            <el-radio label="system">系统规则</el-radio>
            <el-radio label="user">用户规则</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称:">
          <el-input v-model="tempGroup.name" placeholder="请输入规则组名称" />
        </el-form-item>
        <el-form-item label="备注:">
          <el-input v-model="tempGroup.remark" placeholder="请输入备注" />
        </el-form-item>
        <el-form-item label="规则列表:">
          <el-button type="primary" plain size="small" @click="handleAddRule">新增规则</el-button>
          <el-table :data="tempGroup.rules" border style="width: 100%; margin-top: 10px;" size="small">
             <el-table-column prop="matcher_name" label="匹配器" />
             <el-table-column prop="filter1_name" label="过滤器1" />
             <el-table-column prop="action" label="动作" />
             <el-table-column label="操作" width="60">
                <template #default="{$index}">
                    <el-button link type="danger" @click="removeRule($index)">删除</el-button>
                </template>
             </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" @click="saveGroup">确定</el-button>
        </div>
      </template>

      <!-- Nested Dialog: Add Rule for Group -->
      <el-dialog width="600px" v-model="innerVisible" title="新增规则" append-to-body>
        <el-form :model="tempRule" label-width="100px">
           <el-form-item label="匹配器:">
             <el-select v-model="tempRule.matcher_id" placeholder="请选择匹配器" style="width: 100%">
               <el-option v-for="item in matchers" :key="item.id" :label="item.name" :value="item.id" />
             </el-select>
           </el-form-item>
           <el-form-item label="过滤器1:">
             <el-select v-model="tempRule.filter1_id" placeholder="请选择过滤器1" style="width: 100%">
               <el-option v-for="item in filters" :key="item.id" :label="item.name" :value="item.id" />
             </el-select>
           </el-form-item>
           <el-form-item label="动作:">
             <el-select v-model="tempRule.action" placeholder="请选择动作" style="width: 100%">
               <el-option label="拉黑" value="block" />
               <el-option label="放行" value="allow" />
             </el-select>
           </el-form-item>
        </el-form>
        <template #footer>
            <div class="dialog-footer">
              <el-button @click="innerVisible = false">取消</el-button>
              <el-button type="primary" @click="confirmAddRule">确定</el-button>
            </div>
        </template>
      </el-dialog>
    </el-dialog>

    <!-- Dialog: Add Matcher (New) -->
    <el-dialog title="添加匹配器" v-model="matcherDialogVisible" width="800px">
        <el-form :model="tempMatcher" label-width="80px">
            <el-form-item label="类型:">
                <el-radio-group v-model="tempMatcher.type">
                    <el-radio label="system">系统规则</el-radio>
                    <el-radio label="user">用户规则</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item label="名称:">
                <el-input v-model="tempMatcher.name" placeholder="请输入名称" />
            </el-form-item>
            <el-form-item label="备注:">
                <el-input v-model="tempMatcher.remark" placeholder="请输入备注" />
            </el-form-item>
            <el-form-item label="规则:">
                <div style="width: 100%">
                    <el-row :gutter="10" style="margin-bottom: 5px; font-weight: bold; color: #606266; font-size:12px;">
                        <el-col :span="6">匹配项</el-col>
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
                                <el-select v-model="rule.operator" placeholder="操作符" size="small">
                                    <el-option label="等于" value="eq" />
                                    <el-option label="包含" value="contains" />
                                    <el-option label="正则匹配" value="regex" />
                                </el-select>
                            </el-col>
                            <el-col :span="10">
                                <el-input v-model="rule.value" placeholder="一行一个" type="textarea" :rows="1" size="small" />
                            </el-col>
                            <el-col :span="4">
                                <el-button type="primary" link @click="addMatcherRule">添加</el-button>
                                <el-button type="danger" link @click="removeMatcherRule(index)" v-if="tempMatcher.rules.length > 1">删除</el-button>
                            </el-col>
                         </el-row>
                    </div>
                </div>
                <div style="font-size: 12px; color: #999; margin-top: 5px;">
                    多个匹配条件的关系为且，即所有条件都满足时才执行下面的过滤
                </div>
            </el-form-item>
            <el-form-item label="启用:">
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

  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search, ArrowDown, Select } from '@element-plus/icons-vue'
import request from '@/utils/request' // Custom Axios instance

const activeTab = ref('cc')
const ccActiveTab = ref('groups')
const groupsList = ref([])
const matchers = ref([])
const filters = ref([])

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

const dialogFormVisible = ref(false)
const innerVisible = ref(false)
const matcherDialogVisible = ref(false)
const dialogStatus = ref('')
const textMap = {
  update: '编辑规则组',
  create: '添加规则组'
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
    action: 'block',
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

const fetchGroups = async () => {
    try {
        const { data } = await request.get('/admin/rules/cc/groups', { params: listQuery })
        groupsList.value = data.list
    } catch(e) {
        console.error(e)
    }
}

const fetchMatchers = async () => {
    const { data } = await request.get('/admin/rules/cc/matchers', { params: matcherListQuery })
    matchers.value = data.list
}
const fetchFilters = async () => {
    const { data } = await request.get('/admin/rules/cc/filters', { params: filterListQuery })
    filters.value = data.list
}

const handleCreateGroup = () => {
    dialogStatus.value = 'create'
    dialogFormVisible.value = true
    Object.assign(tempGroup, { id: undefined, type: 'system', name: '', remark: '', rules: [] })
}

const handleEditGroup = async (row) => {
    dialogStatus.value = 'update'
    dialogFormVisible.value = true
    // Fetch detailed info including rules
    const { data } = await request.get(`/admin/rules/cc/groups/${row.id}`)
    Object.assign(tempGroup, data)
}

const saveGroup = () => {
    // Mock save
    dialogFormVisible.value = false
    fetchGroups()
}

const handleAddRule = async () => {
    if (matchers.value.length === 0) await fetchMatchers()
    if (filters.value.length === 0) await fetchFilters()
    
    // Reset tempRule
    Object.assign(tempRule, {
        matcher_id: undefined, matcher_name: '',
        filter1_id: undefined, filter1_name: '',
        action: 'block'
    })
    innerVisible.value = true
}

const confirmAddRule = () => {
    // Fill names based on IDs
    const m = matchers.value.find(i => i.id === tempRule.matcher_id)
    if (m) tempRule.matcher_name = m.name
    
    const f1 = filters.value.find(i => i.id === tempRule.filter1_id)
    if (f1) tempRule.filter1_name = f1.name

    tempGroup.rules.push({ ...tempRule })
    innerVisible.value = false
}

const removeRule = (index) => {
    tempGroup.rules.splice(index, 1)
}

const handleCreateMatcher = () => {
    matcherDialogVisible.value = true
    Object.assign(tempMatcher, {
        type: 'system', name: '', remark: '', is_on: true,
        rules: [{ item: 'ip', operator: 'eq', value: '' }]
    })
}

const addMatcherRule = () => {
    tempMatcher.rules.push({ item: 'ip', operator: 'eq', value: '' })
}

const removeMatcherRule = (index) => {
    tempMatcher.rules.splice(index, 1)
}

const saveMatcher = () => {
    matcherDialogVisible.value = false
    // Mock save
    fetchMatchers()
}

const formatAction = (val) => {
    const map = { block: '拉黑', allow: '放行', log: '记录日志' }
    return map[val] || val
}
const formatMode = (val) => {
    return val === 'next' ? '继续下一条规则' : '停止匹配'
}

onMounted(() => {
    fetchGroups()
    fetchMatchers()
    fetchFilters()
})
</script>

<style scoped>
.filter-container {
  padding-bottom: 20px;
}
</style>
