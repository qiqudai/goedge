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

  <AppTable :data="list" border fit highlight-current-row persist-key="cc-groups">
    <el-table-column type="selection" width="55" />
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="user" label="用户" width="100">
      <template #default="{row}">{{ row.is_system ? '系统' : row.user }}</template>
    </el-table-column>
    <el-table-column prop="name" label="名称" />
    <el-table-column label="系统" width="80" align="center">
      <template #default="{row}">
        <el-tag type="success" v-if="row.is_system" effect="dark" size="small" circle>&nbsp;</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="开启" width="80" align="center">
      <template #default="{row}">
        <span :class="row.is_on ? 'text-success' : 'text-danger'">{{ row.is_on ? '启用' : '禁用' }}</span>
      </template>
    </el-table-column>
    <el-table-column prop="sort_order" label="排序" width="80" />
    <el-table-column prop="create_time" label="创建时间" width="160" />
    <el-table-column label="操作" width="120" align="center">
      <template #default="{row}">
        <el-button type="primary" link size="small" @click="handleEdit(row)">编辑</el-button>
        <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
      </template>
    </el-table-column>
  </AppTable>

  <!-- Group Form Dialog -->
  <el-dialog :title="dialogMode === 'create' ? '新增规则组' : '编辑规则组'" v-model="dialogVisible" width="700px">
    <el-form :model="form" label-width="100px">
      <el-form-item label="类型">
        <el-radio-group v-model="form.type">
          <el-radio value="system">系统</el-radio>
          <el-radio value="user">用户</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="名称">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" />
      </el-form-item>
      <el-form-item label="规则">
        <el-button type="primary" plain size="small" @click="openRuleDialog">新增规则项</el-button>
        <el-table :data="form.rules" border size="small" style="margin-top: 10px;">
          <el-table-column label="匹配器">
            <template #default="{row}">{{ getMatcherName(row.matcher_id) }}</template>
          </el-table-column>
          <el-table-column label="过滤器">
            <template #default="{row}">{{ getFilterName(row.filter1_id) }}</template>
          </el-table-column>
          <el-table-column prop="action" label="动作" width="100" />
          <el-table-column label="操作" width="80">
            <template #default="{$index}">
              <el-button link type="danger" @click="removeRule($index)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button size="small" @click="dialogVisible = false">取消</el-button>
      <el-button size="small" type="primary" @click="submitForm">确定</el-button>
    </template>

    <!-- Inner Rule Dialog -->
    <el-dialog v-model="ruleDialogVisible" title="规则设置" width="500px" append-to-body>
      <el-form :model="ruleForm" label-width="80px">
        <el-form-item label="匹配器">
          <el-select v-model="ruleForm.matcher_id" style="width: 100%">
            <el-option v-for="m in matchers" :key="m.id" :label="m.name" :value="m.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="过滤器">
          <el-select v-model="ruleForm.filter1_id" style="width: 100%">
            <el-option v-for="f in filters" :key="f.id" :label="f.name" :value="f.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="动作">
          <el-select v-model="ruleForm.action" style="width: 100%">
            <el-option label="阻断" value="block" />
            <el-option label="放行" value="allow" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button size="small" @click="ruleDialogVisible = false">取消</el-button>
        <el-button size="small" type="primary" @click="addRule">确定</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'

const list = ref([])
const query = reactive({ name: '', status: '' })
const dialogVisible = ref(false)
const dialogMode = ref('create')
const form = reactive({ id: 0, type: 'system', name: '', remark: '', rules: [] })

const matchers = ref([])
const filters = ref([])
const ruleDialogVisible = ref(false)
const ruleForm = reactive({ matcher_id: null, filter1_id: null, action: 'block' })

const fetchData = async () => {
  const { data } = await request.get('/rules/cc/groups', { params: query })
  list.value = data.list || []
}

const handleCreate = () => {
  dialogMode.value = 'create'
  Object.assign(form, { id: 0, type: 'system', name: '', remark: '', rules: [] })
  dialogVisible.value = true
}

const handleEdit = async (row) => {
  dialogMode.value = 'update'
  const { data } = await request.get(`/rules/cc/groups/${row.id}`)
  Object.assign(form, data)
  if (!form.rules) form.rules = []
  dialogVisible.value = true
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

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除规则组?', '提示', { type: 'warning' }).then(async () => {
    await request.delete(`/rules/cc/groups/${row.id}`)
    ElMessage.success('删除成功')
    fetchData()
  })
}

const openRuleDialog = async () => {
  if (!matchers.value.length) {
    const mRes = await request.get('/rules/cc/matchers')
    matchers.value = mRes.data.list || []
  }
  if (!filters.value.length) {
    const fRes = await request.get('/rules/cc/filters')
    filters.value = fRes.data.list || []
  }
  Object.assign(ruleForm, { matcher_id: null, filter1_id: null, action: 'block' })
  ruleDialogVisible.value = true
}

const addRule = () => {
  form.rules.push({ ...ruleForm })
  ruleDialogVisible.value = false
}

const removeRule = (idx) => form.rules.splice(idx, 1)

const getMatcherName = (id) => matchers.value.find(m => m.id === id)?.name || id
const getFilterName = (id) => filters.value.find(f => f.id === id)?.name || id

onMounted(fetchData)
</script>
