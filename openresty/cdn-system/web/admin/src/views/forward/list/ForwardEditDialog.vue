<template>
  <el-dialog
    v-model="visible"
    :title="form.id ? '编辑转发' : '添加转发'"
    width="640px"
    @closed="handleClosed"
    class="forward-dialog"
  >
    <el-tabs v-model="activeTab" class="dialog-tabs">
      <el-tab-pane label="单个" name="single" />
      <el-tab-pane v-if="!form.id" label="批量" name="batch" />
    </el-tabs>

    <div class="dialog-content">
      <el-form :model="form" label-width="100px" ref="formRef" :rules="rules">
        <el-form-item v-if="isAdmin" label="用户选择：" prop="user_id">
          <el-select v-model="form.user_id" placeholder="选择用户" style="width: 100%" filterable @change="handleUserChange">
            <el-option v-for="u in users" :key="u.id" :label="`${u.name} (id: ${u.id})`" :value="u.id" />
          </el-select>
        </el-form-item>

        <el-form-item label="用户套餐：" prop="user_package_id">
          <el-select v-model="form.user_package_id" placeholder="请选择" style="width: 100%" clearable>
            <el-option v-for="p in userPackages" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>

        <template v-if="activeTab === 'single'">
            <el-form-item label="监听端口：" prop="listen_ports">
              <el-input v-model="form.listen_ports" placeholder="88、99/udp或88/tcp,多个端口空格分隔" />
            </el-form-item>

            <el-form-item label="源地址:端口：" prop="origin">
              <el-input v-model="form.origin" placeholder="1.1.1.1:99或www.abc.com:99" />
            </el-form-item>
        </template>

        <template v-else>
            <el-form-item label="转发数据：" prop="batch_data">
              <el-input
                v-model="form.batch_data"
                type="textarea"
                :rows="5"
                placeholder="格式为：监听端口|IP|回源端口&#10;88 99/udp|1.2.3.4|8080&#10;77 66|8.8.8.8|8080"
              />
            </el-form-item>
            <el-form-item label="忽略错误：">
               <div style="display: flex; align-items: center; gap: 10px;">
                  <el-switch v-model="form.ignore_errors" />
                  <span class="form-tip-small">有转发添加出错时，不中断，继续添加下一条</span>
               </div>
            </el-form-item>
        </template>

        <div class="expand-divider" @click="isExpanded = !isExpanded">
           <span class="line"></span>
           <span class="text">展开更多 <el-icon><component :is="isExpanded ? 'ArrowUp' : 'ArrowDown'" /></el-icon></span>
           <span class="line"></span>
        </div>

        <div v-show="isExpanded">
            <el-form-item label="所属分组：">
              <div style="display: flex; gap: 8px; width: 100%;">
                  <el-select v-model="form.group_id" placeholder="转发分组，可选" style="flex: 1" clearable>
                    <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
                  </el-select>
                  <el-button @click="showAddGroup = true"><el-icon><Plus /></el-icon></el-button>
              </div>
            </el-form-item>

            <el-form-item label="备注：">
              <el-input v-model="form.remark" placeholder="输入备注信息" />
            </el-form-item>
        </div>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
    </template>

    <!-- Inline Add Group Dialog -->
    <el-dialog v-model="showAddGroup" title="新增转发分组" width="380px" append-to-body>
        <el-form :model="groupForm" label-width="60px">
            <el-form-item label="名称"><el-input v-model="groupForm.name" /></el-form-item>
            <el-form-item label="备注"><el-input v-model="groupForm.remark" /></el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="showAddGroup = false">取消</el-button>
            <el-button type="primary" @click="handleAddGroup">确定</el-button>
        </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowUp, Plus } from '@element-plus/icons-vue'
import request from '@/utils/request'

const props = defineProps({
  modelValue: Boolean,
  data: Object,
  isAdmin: { type: Boolean, default: true }
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const submitting = ref(false)
const formRef = ref(null)
const activeTab = ref('single')
const isExpanded = ref(false)
const showAddGroup = ref(false)

const users = ref([])
const userPackages = ref([])
const groups = ref([])

const form = reactive({
  id: 0,
  user_id: '',
  user_package_id: '',
  listen_ports: '',
  origin: '',
  batch_data: '',
  ignore_errors: false,
  group_id: '',
  remark: ''
})

const groupForm = reactive({ name: '', remark: '' })

const rules = {
  user_id: [{ required: true, message: '请选择用户', trigger: 'change' }],
  user_package_id: [{ required: true, message: '请选择套餐', trigger: 'change' }],
  listen_ports: [{ required: true, message: '请输入监听端口', trigger: 'blur' }],
  origin: [{ required: true, message: '请输入源地址:端口', trigger: 'blur' }],
  batch_data: [{ required: true, message: '请输入批量转发数据', trigger: 'blur' }]
}

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    initData()
    if (props.data) {
      Object.assign(form, props.data)
      activeTab.value = 'single'
    } else {
      resetForm()
    }
  }
})

watch(() => visible.value, (val) => {
  emit('update:modelValue', val)
})

const initData = async () => {
  try {
    const [uRes, gRes] = await Promise.all([
      props.isAdmin ? request.get('/users') : Promise.resolve({ data: { list: [] } }),
      request.get('/forward_groups')
    ])
    if (props.isAdmin) users.value = uRes.data?.list || []
    groups.value = gRes.data?.list || []
    
    if (form.user_id) {
        handleUserChange(form.user_id)
    }
  } catch (e) {
    console.error('Failed to init dialog data', e)
  }
}

const handleUserChange = async (userId) => {
    if (!userId) {
        userPackages.value = []
        return
    }
    const res = await request.get('/user_packages', { params: { user_id: userId } })
    userPackages.value = res.data?.list || []
}

const handleAddGroup = async () => {
    if (!groupForm.name) return
    await request.post('/forward_groups', groupForm)
    ElMessage.success('分组添加成功')
    showAddGroup.value = false
    groupForm.name = ''
    groupForm.remark = ''
    const gRes = await request.get('/forward_groups')
    groups.value = gRes.data?.list || []
}

const resetForm = () => {
  Object.assign(form, {
    id: 0,
    user_id: '',
    user_package_id: '',
    listen_ports: '',
    origin: '',
    batch_data: '',
    ignore_errors: false,
    group_id: '',
    remark: ''
  })
  activeTab.value = 'single'
  isExpanded.value = false
}

const handleClosed = () => {
  formRef.value?.resetFields()
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  submitting.value = true
  try {
    const payload = { ...form }
    if (activeTab.value === 'batch') {
        // Handle batch mapping logic if necessary on backend
    }
    
    if (form.id) {
      await request.put(`/forwards/${form.id}`, payload)
    } else {
      if (activeTab.value === 'batch') {
        const batchPayload = {
          user_id: form.user_id,
          user_package_id: form.user_package_id,
          group_id: form.group_id,
          data: form.batch_data,
          ignore_error: form.ignore_errors,
          remark: form.remark
        }
        await request.post('/forwards/batch', batchPayload)
      } else {
        const singlePayload = {
          user_id: form.user_id,
          user_package_id: form.user_package_id,
          group_id: form.group_id,
          listen_ports_input: form.listen_ports,
          origin_input: form.origin,
          remark: form.remark
        }
        await request.post('/forwards', singlePayload)
      }
    }
    ElMessage.success('操作成功')
    emit('success')
    visible.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<script>
export default {
  inheritAttrs: false
}
</script>

<style scoped>
.forward-dialog :deep(.el-dialog__body) { padding-top: 5px; }
.dialog-tabs { margin-bottom: 20px; }
.dialog-content { padding: 0 10px; }
.expand-divider { 
    display: flex; 
    align-items: center; 
    justify-content: center; 
    margin: 24px 0; 
    cursor: pointer;
    user-select: none;
    color: #909399;
    font-size: 13px;
}
.expand-divider .line { flex: 1; height: 1px; background: #ebeef5; border-bottom: 1px dashed #ebeef5; }
.expand-divider .text { padding: 0 15px; display: flex; align-items: center; gap: 5px; }
.form-tip-small { font-size: 12px; color: #909399; }
</style>

