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
        <el-alert v-if="!canCreateWithPackage" :title="noPackageMessage" type="warning" :closable="false" class="package-alert" />

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
                <SiteGroupSelect v-model="form.group_ids" type="forward" :user-id="form.user_id" multiple />
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

  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import request from '@/utils/request'
import SiteGroupSelect from '@/components/SiteGroupSelect.vue'

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

const users = ref([])
const userPackages = ref([])
const lastUserId = ref(0)
const noPackageMessage = '当前用户没有可用套餐，不能添加 L4 转发'
const canCreateWithPackage = computed(() => form.id || userPackages.value.length > 0)

const form = reactive({
  id: 0,
  user_id: '',
  user_package_id: '',
  listen_ports: '',
  origin: '',
  batch_data: '',
  ignore_errors: false,
  group_ids: [],
  remark: ''
})

const rules = computed(() => {
  const baseRules = {
    user_id: [{ required: true, message: '请选择用户', trigger: 'change' }],
    user_package_id: [{ required: true, message: '请选择套餐', trigger: 'change' }],
    listen_ports: [{ required: true, message: '请输入监听端口', trigger: 'blur' }],
    origin: [{ required: true, message: '请输入源地址:端口', trigger: 'blur' }],
    batch_data: [{ required: true, message: '请输入批量转发数据', trigger: 'blur' }]
  }
  if (!props.isAdmin) {
    delete baseRules.user_id
  }
  return baseRules
})

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    if (props.data) {
      const data = { ...props.data }
      data.group_ids = data.group_ids || (data.group_id ? [data.group_id] : [])
      Object.assign(form, {
        ...data,
        user_id: Number(data.user_id) || 0,
        user_package_id: Number(data.user_package_id) || 0
      })
      activeTab.value = 'single'
    } else {
      resetForm()
    }
    initData()
  }
})

watch(() => visible.value, (val) => {
  emit('update:modelValue', val)
})

const initData = async () => {
  try {
    if (props.isAdmin) {
      const uRes = await request.get('/users')
      users.value = uRes.data?.list || []
      if (form.user_id) {
        await handleUserChange(form.user_id)
      }
      return
    }
    await loadSelfPackages()
  } catch (e) {
    console.error('Failed to init dialog data', e)
  }
}

const ensureDefaultPackage = () => {
  if (!form.user_package_id && userPackages.value.length > 0) {
    form.user_package_id = userPackages.value[0].id
  }
}

const handleUserChange = async (userId) => {
  if (!userId) {
    userPackages.value = []
    lastUserId.value = 0
    form.user_package_id = 0
    return
  }
  const resolvedId = Number(userId) || 0
  if (resolvedId !== lastUserId.value) {
    form.user_package_id = 0
    form.group_ids = []
    lastUserId.value = resolvedId
  }
  const res = await request.get('/user_packages', { params: { user_id: resolvedId, pageSize: 1000 } })
  userPackages.value = res.data?.list || res.list || []
  ensureDefaultPackage()
}

const loadSelfPackages = async () => {
  const res = await request.get('/user_packages', { params: { pageSize: 1000 } })
  userPackages.value = res.data?.list || res.list || []
  ensureDefaultPackage()
}

// handleAddGroup removed

const resetForm = () => {
  Object.assign(form, {
    id: 0,
    user_id: 0,
    user_package_id: 0,
    listen_ports: '',
    origin: '',
    batch_data: '',
    ignore_errors: false,
    group_ids: [],
    remark: ''
  })
  activeTab.value = 'single'
  isExpanded.value = false
  lastUserId.value = 0
}

const handleClosed = () => {
  formRef.value?.resetFields()
}

const handleSubmit = async () => {
  if (!form.id && userPackages.value.length === 0) {
    ElMessage.error(noPackageMessage)
    return
  }
  if (!form.user_package_id) {
    ElMessage.error('请选择套餐')
    return
  }
  await formRef.value?.validate()
  submitting.value = true
  try {
    const basePayload = {
      user_package_id: Number(form.user_package_id) || 0,
      group_ids: Array.isArray(form.group_ids) ? form.group_ids : [],
      remark: form.remark
    }
    if (props.isAdmin) {
      basePayload.user_id = Number(form.user_id) || 0
    }

    if (form.id) {
      const updatePayload = {
        ...basePayload,
        listen_ports_input: form.listen_ports,
        origin_input: form.origin
      }
      await request.put(`/forwards/${form.id}`, updatePayload)
    } else if (activeTab.value === 'batch') {
      const batchPayload = {
        ...basePayload,
        data: form.batch_data,
        ignore_error: form.ignore_errors
      }
      await request.post('/forwards/batch', batchPayload)
    } else {
      const singlePayload = {
        ...basePayload,
        listen_ports_input: form.listen_ports,
        origin_input: form.origin
      }
      await request.post('/forwards', singlePayload)
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
.package-alert { margin: -8px 0 14px 100px; width: calc(100% - 100px); }
.expand-divider { 
    display: flex; 
    align-items: center; 
    justify-content: center; 
    margin: 24px 0; 
    cursor: pointer;
    user-select: none;
    color: var(--el-text-color-secondary);
    font-size: 13px;
}
.expand-divider .line { flex: 1; height: 1px; background: var(--el-border-color-lighter); border-bottom: 1px dashed var(--el-border-color-lighter); }
.expand-divider .text { padding: 0 15px; display: flex; align-items: center; gap: 5px; }
.form-tip-small { font-size: 12px; color: var(--el-text-color-secondary); }
</style>

