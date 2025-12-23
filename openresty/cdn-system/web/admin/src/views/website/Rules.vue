<template>
  <div class="app-container">
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane label="CC??" name="cc">
        <el-tabs v-model="ccActiveTab">
          <el-tab-pane label="???" name="groups">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateGroup">?????</el-button>
              <el-select v-model="listQuery.status" placeholder="??" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="??" value="on" />
                <el-option label="??" value="off" />
              </el-select>
              <el-input v-model="listQuery.name" placeholder="?????, ????" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchGroups">??</el-button>
            </div>

            <el-table :data="groupsList" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="??" width="100">
                <template #default="{row}">{{ row.is_system ? '??' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="??" />
              <el-table-column label="????" width="100" align="center">
                <template #default="{row}">
                  <el-tag type="success" v-if="row.is_system" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="??" width="100" align="center">
                <template #default="{row}">
                  <el-tag type="success" v-if="row.is_show" effect="dark" size="small" style="border-radius: 50%; width: 20px; height: 20px; padding: 0;">&nbsp;</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="??" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '??' : '??' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="sort_order" label="??" width="80" />
              <el-table-column prop="create_time" label="????" width="160" />
              <el-table-column label="??" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small" @click="handleEditGroup(row)">??</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="???" name="matchers">
            <div class="filter-container">
              <el-button type="primary" class="filter-item" @click="handleCreateMatcher">?????</el-button>
              <el-select v-model="matcherListQuery.status" placeholder="??" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="??" value="on" />
                <el-option label="??" value="off" />
              </el-select>
              <el-input v-model="matcherListQuery.name" placeholder="??" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchMatchers">??</el-button>
            </div>

            <el-table :data="matchers" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="??" width="100">
                <template #default="{row}">{{ row.is_system ? '??' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="??" />
              <el-table-column label="????" width="100" align="center">
                <template #default="{row}">
                  <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column label="??" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '??' : '??' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="????" width="160" />
              <el-table-column label="??" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small">??</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane label="???" name="filters">
            <div class="filter-container">
              <el-button type="primary" class="filter-item">?????</el-button>
              <el-select v-model="filterListQuery.status" placeholder="??" class="filter-item" style="width: 120px; margin-left:10px;">
                <el-option label="??" value="on" />
                <el-option label="??" value="off" />
              </el-select>
              <el-input v-model="filterListQuery.name" placeholder="??" style="width: 200px; margin-left: 10px;" class="filter-item" />
              <el-button class="filter-item" type="primary" :icon="Search" @click="fetchFilters">??</el-button>
            </div>

            <el-table :data="filters" border fit highlight-current-row style="width: 100%">
              <el-table-column type="selection" width="55" />
              <el-table-column prop="id" label="ID" width="80" />
              <el-table-column prop="user" label="??" width="100">
                <template #default="{row}">{{ row.is_system ? '??' : row.user }}</template>
              </el-table-column>
              <el-table-column prop="name" label="??" />
              <el-table-column label="????" width="100" align="center">
                <template #default="{row}">
                  <el-icon v-if="row.is_system" color="#67C23A"><Select /></el-icon>
                </template>
              </el-table-column>
              <el-table-column prop="type" label="??" width="150" />
              <el-table-column label="??" width="100" align="center">
                <template #default="{row}">
                  <span :style="{ color: row.is_on ? '#67C23A' : '#F56C6C' }">{{ row.is_on ? '??' : '??' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="create_time" label="????" width="160" />
              <el-table-column label="??" width="150" align="center">
                <template #default="{row}">
                  <el-button type="primary" link size="small">??</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-tab-pane>

      <el-tab-pane label="ACL??" name="acl">
        <div class="filter-container">
          <el-button type="primary" class="filter-item" @click="openAclDialog()">??ACL</el-button>
          <el-select v-model="aclQuery.status" placeholder="??" class="filter-item" style="width: 120px; margin-left:10px;">
            <el-option label="??" value="on" />
            <el-option label="??" value="off" />
          </el-select>
          <el-input v-model="aclQuery.name" placeholder="??" style="width: 200px; margin-left: 10px;" class="filter-item" />
          <el-button class="filter-item" type="primary" :icon="Search" @click="fetchAcl">??</el-button>
        </div>

        <el-table :data="aclList" border fit highlight-current-row style="width: 100%">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="??" min-width="160" />
          <el-table-column prop="default_action" label="????" width="120">
            <template #default="{row}">
              {{ row.default_action === 'deny' ? '??' : '??' }}
            </template>
          </el-table-column>
          <el-table-column prop="enable" label="??" width="100">
            <template #default="{row}">
              <el-tag :type="row.enable ? 'success' : 'info'">{{ row.enable ? '??' : '??' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="create_time" label="????" width="160" />
          <el-table-column label="??" width="160" align="center">
            <template #default="{row}">
              <el-button link type="primary" size="small" @click="openAclDialog(row)">??</el-button>
              <el-button link type="danger" size="small" @click="deleteAcl(row)">??</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog :title="textMap[dialogStatus]" v-model="dialogFormVisible" width="800px">
      <el-form :model="tempGroup" label-position="right" label-width="100px" style="width: 700px; margin-left:50px;">
        <el-form-item label="??">
          <el-radio-group v-model="tempGroup.type">
            <el-radio label="system">????</el-radio>
            <el-radio label="user">????</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model="tempGroup.name" placeholder="????????" />
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model="tempGroup.remark" placeholder="?????" />
        </el-form-item>
        <el-form-item label="????">
          <el-button type="primary" plain size="small" @click="handleAddRule">????</el-button>
          <el-table :data="tempGroup.rules" border style="width: 100%; margin-top: 10px;" size="small">
            <el-table-column prop="matcher_name" label="???" />
            <el-table-column prop="filter1_name" label="???" />
            <el-table-column prop="action" label="??" />
            <el-table-column label="??" width="60">
              <template #default="{$index}">
                <el-button link type="danger" @click="removeRule($index)">??</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">??</el-button>
          <el-button type="primary" @click="saveGroup">??</el-button>
        </div>
      </template>

      <el-dialog width="600px" v-model="innerVisible" title="????" append-to-body>
        <el-form :model="tempRule" label-width="100px">
          <el-form-item label="???">
            <el-select v-model="tempRule.matcher_id" placeholder="??????" style="width: 100%">
              <el-option v-for="item in matchers" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="???">
            <el-select v-model="tempRule.filter1_id" placeholder="??????" style="width: 100%">
              <el-option v-for="item in filters" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="??">
            <el-select v-model="tempRule.action" placeholder="?????" style="width: 100%">
              <el-option label="??" value="block" />
              <el-option label="??" value="allow" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <div class="dialog-footer">
            <el-button @click="innerVisible = false">??</el-button>
            <el-button type="primary" @click="confirmAddRule">??</el-button>
          </div>
        </template>
      </el-dialog>
    </el-dialog>

    <el-dialog title="?????" v-model="matcherDialogVisible" width="800px">
      <el-form :model="tempMatcher" label-width="80px">
        <el-form-item label="??">
          <el-radio-group v-model="tempMatcher.type">
            <el-radio label="system">????</el-radio>
            <el-radio label="user">????</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model="tempMatcher.name" placeholder="?????" />
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model="tempMatcher.remark" placeholder="?????" />
        </el-form-item>
        <el-form-item label="??">
          <div style="width: 100%">
            <el-row :gutter="10" style="margin-bottom: 5px; font-weight: bold; color: #606266; font-size:12px;">
              <el-col :span="6">???</el-col>
              <el-col :span="4">???</el-col>
              <el-col :span="10">???</el-col>
              <el-col :span="4">??</el-col>
            </el-row>
            <div v-for="(rule, index) in tempMatcher.rules" :key="index" style="margin-bottom: 10px;">
              <el-row :gutter="10">
                <el-col :span="6">
                  <el-select v-model="rule.item" placeholder="???" size="small">
                    <el-option label="IP??" value="ip" />
                    <el-option label="URL" value="url" />
                    <el-option label="User-Agent" value="ua" />
                    <el-option label="Referer" value="referer" />
                  </el-select>
                </el-col>
                <el-col :span="4">
                  <el-select v-model="rule.operator" placeholder="???" size="small">
                    <el-option label="??" value="eq" />
                    <el-option label="??" value="contains" />
                    <el-option label="????" value="regex" />
                  </el-select>
                </el-col>
                <el-col :span="10">
                  <el-input v-model="rule.value" placeholder="????" type="textarea" :rows="1" size="small" />
                </el-col>
                <el-col :span="4">
                  <el-button type="primary" link @click="addMatcherRule">??</el-button>
                  <el-button type="danger" link @click="removeMatcherRule(index)" v-if="tempMatcher.rules.length > 1">??</el-button>
                </el-col>
              </el-row>
            </div>
          </div>
          <div style="font-size: 12px; color: #999; margin-top: 5px;">
            ?????????? AND??????????????
          </div>
        </el-form-item>
        <el-form-item label="??">
          <el-switch v-model="tempMatcher.is_on" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="matcherDialogVisible = false">??</el-button>
          <el-button type="primary" @click="saveMatcher">??</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="aclDialogVisible" :title="aclDialogTitle" width="680px">
      <el-form :model="aclForm" label-width="100px">
        <el-form-item label="??">
          <el-input v-model="aclForm.name" placeholder="?????" />
        </el-form-item>
        <el-form-item label="??">
          <el-input v-model="aclForm.des" placeholder="?????" />
        </el-form-item>
        <el-form-item label="????">
          <el-radio-group v-model="aclForm.default_action">
            <el-radio label="allow">??</el-radio>
            <el-radio label="deny">??</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="??">
          <el-switch v-model="aclForm.enable" />
        </el-form-item>
        <el-form-item label="??">
          <el-button type="primary" plain size="small" @click="addAclRule">????</el-button>
          <el-table :data="aclForm.rules" border style="width: 100%; margin-top: 10px;" size="small">
            <el-table-column label="IP">
              <template #default="{ row }">
                <el-input v-model="row.ip" placeholder="IP?CIDR" />
              </template>
            </el-table-column>
            <el-table-column label="??" width="140">
              <template #default="{ row }">
                <el-select v-model="row.action" style="width: 100%;">
                  <el-option label="??" value="allow" />
                  <el-option label="??" value="deny" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="??" width="80">
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeAclRule($index)">??</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="aclDialogVisible = false">??</el-button>
        <el-button type="primary" @click="saveAcl">??</el-button>
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
  update: '?????',
  create: '?????'
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
const aclDialogTitle = computed(() => (aclForm.id ? '??ACL' : '??ACL'))

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
  ElMessage.success('????')
  aclDialogVisible.value = false
  fetchAcl()
}

const deleteAcl = row => {
  ElMessageBox.confirm('?????ACL?', '??', {
    confirmButtonText: '??',
    cancelButtonText: '??',
    type: 'warning'
  }).then(() => {
    request.delete(`/rules/acl/${row.id}`).then(() => {
      ElMessage.success('????')
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
