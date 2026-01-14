<template>
  <el-dialog
    v-model="visible"
    :title="form.id ? '编辑网站' : '添加网站'"
    width="680px"
    @closed="handleClosed"
    :close-on-click-modal="false"
  >
    <el-tabs v-if="!form.id" v-model="activeTab" type="card">
      <el-tab-pane label="单个添加" name="single" />
      <el-tab-pane label="批量添加" name="batch" />
    </el-tabs>

    <div style="margin-top: 10px;">
      <!-- Single Mode Form -->
      <el-form v-if="activeTab === 'single'" :model="form" label-width="120px" ref="singleFormRef" :rules="rules">
        <el-form-item v-if="isAdmin" label="所属用户" prop="user_id">
          <el-select 
            v-model="form.user_id" 
            placeholder="搜索用户 (默认管理员)" 
            style="width: 100%" 
            filterable
            automatic-dropdown
            :loading="userLoading"
            @visible-change="handleUserDropdown"
            @change="handleUserChange"
            clearable
          >
            <el-option v-for="u in users" :key="u.id" :label="u.name + ' (' + u.username + ')'" :value="u.id" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="网站域名" prop="domains">
          <el-input
            v-model="form.domains"
            placeholder="每行一个域名，支持泛域名如 *.example.com"
          />
        </el-form-item>
        <el-alert
          v-if="domainUsage"
          :type="domainLimitExceeded ? 'error' : 'info'"
          :closable="false"
          class="limit-alert"
        >
          域名数 {{ domainUsage.total_domains }} / {{ formatLimit(domainUsage.domain_limit) }} 主域名数 {{ domainUsage.total_main_domains }} / {{ formatLimit(domainUsage.main_domain_limit) }}
        </el-alert>

        <el-form-item label="源站地址" prop="origins">
          <el-input
            v-model="form.origins"
            placeholder="每行一个，如: 1.1.1.1 或 1.1.1.1:8080"
          />
        </el-form-item>

        <el-form-item label="网站套餐" prop="user_package_id">
          <el-select v-model="form.user_package_id" placeholder="选择套餐 (可选)" style="width: 100%" clearable>
            <el-option v-for="p in userPackages" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>

        <el-form-item label="加速类型">
           <el-radio-group v-model="form.site_type">
             <el-radio value="website">网页加速</el-radio>
             <el-radio value="api">API加速</el-radio>
             <el-radio value="download">下载加速</el-radio>
           </el-radio-group>
        </el-form-item>

        <div class="expand-more" @click="expandMore = !expandMore">
            <span>{{ expandMore ? '收起更多' : '展开更多' }}</span>
            <el-icon><component :is="expandMore ? 'ArrowUp' : 'ArrowDown'" /></el-icon>
        </div>

        <div v-show="expandMore" class="extra-fields">
            <el-form-item label="网站分组">
                <SiteGroupSelect
                  v-model="form.group_ids"
                  :user-id="form.user_id"
                  multiple
                  :key="`single-${form.user_id || 'self'}`"
                  ref="singleGroupRef"
                />
            </el-form-item>
            <el-form-item label="DNS 接口">
                <el-select v-model="form.dns_provider_id" placeholder="自动添加解析记录 (可选)" style="width: 100%" clearable>
                <el-option v-for="d in dnsProviders" :key="d.id" :label="d.name" :value="d.id" />
                </el-select>
            </el-form-item>
            <el-form-item label="备注">
                <el-input v-model="form.remark" placeholder="可选备注信息" />
            </el-form-item>
        </div>
      </el-form>

      <div v-if="activeTab === 'batch'">
        <div v-if="!currentBatchId">
            <el-form :model="batchForm" label-width="120px" ref="batchFormRef">
                <el-form-item v-if="isAdmin" label="所属用户">
                <el-select 
                    v-model="batchForm.user_id" 
                    placeholder="搜索用户" 
                    style="width: 100%" 
                    filterable
                    automatic-dropdown
                    :loading="userLoading"
                    @visible-change="handleUserDropdown"
                    @change="handleBatchUserChange"
                    clearable
                >
                    <el-option v-for="u in users" :key="u.id" :label="u.name" :value="u.id" />
                </el-select>
                </el-form-item>
                <el-form-item label="网站套餐">
                <el-select v-model="batchForm.user_package_id" placeholder="选择套餐" style="width: 100%" clearable>
                    <el-option v-for="p in userPackages" :key="p.id" :label="p.name" :value="p.id" />
                </el-select>
                </el-form-item>
                
                <el-tabs v-model="batchMode" type="border-card" class="mb-4">
                    <el-tab-pane label="简单模式" name="simple">
                        <el-form-item label="网站域名" required>
                            <DomainBatchInput v-model="batchForm.simpleDomains" @validation="handleBatchValidation" />
                        </el-form-item>
                        <el-form-item label="源站地址" required>
                             <el-input 
                                v-model="batchForm.simpleBackends" 
                                type="textarea" 
                                :rows="3" 
                                placeholder="所有域名共享的源站，每行一个。如: 1.1.1.1" 
                             />
                        </el-form-item>
                    </el-tab-pane>
                    <el-tab-pane label="高级模式" name="advanced">
                         <el-form-item label="数据内容" required>
                            <el-input
                                v-model="batchForm.data"
                                type="textarea"
                                :rows="8"
                                placeholder="格式: domain=域名|ip=源IP
示例:
domain=example.com,www.example.com|ip=1.1.1.1
domain=test.com|ip=2.2.2.2,3.3.3.3"
                            />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>

                <el-form-item label="选项">
                    <el-checkbox v-model="batchForm.ignore_error">忽略错误</el-checkbox>
                </el-form-item>
                
                <el-form-item label="网站分组">
                    <SiteGroupSelect
                      v-model="batchForm.group_id"
                      :user-id="batchForm.user_id"
                      :key="`batch-${batchForm.user_id || 'self'}`"
                      ref="batchGroupRef"
                    />
                </el-form-item>
                
                <el-form-item label="DNS 接口">
                    <el-select v-model="batchForm.dns_provider_id" placeholder="自动添加解析记录 (可选)" style="width: 100%" clearable>
                        <el-option v-for="d in dnsProviders" :key="d.id" :label="d.name" :value="d.id" />
                    </el-select>
                </el-form-item>
            </el-form>
        </div>
      </div>
    </div>

    <template #footer>
      <div>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" :disabled="submitting" @click="handleSubmit">确定</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import request from '@/utils/request'

import SiteGroupSelect from '@/components/SiteGroupSelect.vue'
import DomainBatchInput from '@/components/DomainBatchInput.vue'

const props = defineProps({
  modelValue: Boolean,
  data: Object,
  isAdmin: Boolean
})

const emit = defineEmits(['update:modelValue', 'success'])

const visible = ref(false)
const submitting = ref(false)
const activeTab = ref('single')
const expandMore = ref(false)
const singleFormRef = ref(null)
const batchFormRef = ref(null)

const users = ref([])
const userLoading = ref(false)
const userPackages = ref([])
const dnsProviders = ref([])
const lastUserId = ref('')
const singleGroupRef = ref(null)
const batchGroupRef = ref(null)

const form = reactive({
  id: 0,
  user_id: '',
  domains: '',
  origins: '',
  user_package_id: '',
  group_ids: [],
  group_id: '', // keep for legacy compatibility if needed
  dns_provider_id: '',
  site_type: 'website',
  remark: ''
})

const batchForm = reactive({
  user_id: '',
  user_package_id: '',
  group_id: '',
  dns_provider_id: '',
  data: '',
  ignore_error: false,
  simpleDomains: '',
  simpleBackends: ''
})

const batchMode = ref('simple')
const currentBatchId = ref('')
const validBatchDomains = ref([])
const domainUsage = ref(null)
const domainLimitExceeded = ref(false)

const rules = {
  domains: [
    { required: true, message: '请输入域名', trigger: 'blur' },
    { 
      validator: (rule, value, callback) => {
        if (!value) return callback(new Error('请输入域名'))
        const lines = value.split('\n').map(s => s.trim()).filter(Boolean)
        const domainRegex = /^(?:\*\.)?(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
        for (const line of lines) {
           if (!domainRegex.test(line) && !/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(line)) {
               return callback(new Error(`"${line}" 不是有效的域名格式`))
           }
        }
        callback()
      }, 
      trigger: 'blur' 
    }
  ],
  origins: [{ required: true, message: '请输入源站地址', trigger: 'blur' }],
}

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    if (props.data) {
      activeTab.value = 'single'
      // Editing mode
      const data = { ...props.data }
      if (Array.isArray(data.domains)) data.domains = data.domains.join('\n')
      if (Array.isArray(data.origins)) data.origins = data.origins.map(o => o.address || o).join('\n')
      // Map legacy fields if necessary
      Object.assign(form, {
          id: data.id,
          user_id: data.user_id,
          domains: data.domains,
          origins: data.origins,
          user_package_id: data.user_package_id,
          group_ids: data.group_ids || (data.group_id ? [data.group_id] : []),
          group_id: data.group_id, // Need to make sure relation is loaded
          dns_provider_id: data.dns_provider_id,
          site_type: data.type || 'website', // assuming 'type' field
          remark: data.remark
      })
      
      // Load dependencies for this user
      if (data.user_id) handleUserChange(data.user_id)
    } else {
      activeTab.value = 'single'
      resetForm()
    }
    loadCommonData()
  }
})

watch(() => visible.value, (val) => {
  emit('update:modelValue', val)
})

watch(() => [form.user_id, form.user_package_id], () => {
  if (activeTab.value === 'single') {
    fetchDomainUsage()
  }
})

watch(() => activeTab.value, () => {
  if (activeTab.value === 'single') {
    fetchDomainUsage()
  }
})

const loadCommonData = async () => {
    // Load things that don't depend on user
    if (props.isAdmin) {
        searchUsers('') 
    } else {
        await loadSelfData()
    }
}

const searchUsers = async (query) => {
    if (!props.isAdmin) return
    userLoading.value = true
    try {
        const res = await request.get('/users', { params: { keyword: query, pageSize: 200 } })
        users.value = (res.data?.list || []).map(u => ({...u, username: u.username || u.email}))
    } finally {
        userLoading.value = false
    }
}

const handleUserDropdown = (visible) => {
    if (!visible || !props.isAdmin) return
    if (!users.value.length) {
        searchUsers('')
    }
}

const reloadGroupSelect = async () => {
  await nextTick()
  singleGroupRef.value?.reload?.()
  batchGroupRef.value?.reload?.()
}

const handleUserChange = async (uid) => {
    if (!uid) {
        userPackages.value = []
        dnsProviders.value = []
        form.group_ids = []
        batchForm.group_id = ''
        lastUserId.value = ''
        await reloadGroupSelect()
        return
    }

    if (String(uid) !== String(lastUserId.value)) {
        form.group_ids = []
        batchForm.group_id = ''
        lastUserId.value = uid
    }
    
    try {
        const [pkgRes, dnsRes] = await Promise.all([
            request.get('/user_packages', { params: { user_id: uid, pageSize: 1000 } }),
            request.get('/dnsapi', { params: { user_id: uid, pageSize: 1000 } })
        ])

        // safe data mapping
        userPackages.value = pkgRes.data?.list || pkgRes.list || []
        dnsProviders.value = dnsRes.data?.list || dnsRes.list || []

        // 1. Auto-select first package
        if (userPackages.value.length > 0) {
            const firstPkgId = userPackages.value[0].id
            if (activeTab.value === 'single' && !form.user_package_id) {
                form.user_package_id = firstPkgId
            } else if (activeTab.value === 'batch' && !batchForm.user_package_id) {
                batchForm.user_package_id = firstPkgId
            }
        }
    } catch (e) {
        userPackages.value = []
        dnsProviders.value = []
    } finally {
        await reloadGroupSelect()
    }
}

const handleBatchUserChange = (uid) => {
    handleUserChange(uid)
}

const loadSelfData = async () => {
    try {
        const [pkgRes, dnsRes] = await Promise.all([
            request.get('/user_packages', { params: { pageSize: 1000 } }),
            request.get('/dnsapi', { params: { pageSize: 1000 } })
        ])
        userPackages.value = pkgRes.data?.list || pkgRes.list || []
        dnsProviders.value = dnsRes.data?.list || dnsRes.list || []

        if (userPackages.value.length > 0) {
            const firstPkgId = userPackages.value[0].id
            if (activeTab.value === 'single' && !form.user_package_id) {
                form.user_package_id = firstPkgId
            } else if (activeTab.value === 'batch' && !batchForm.user_package_id) {
                batchForm.user_package_id = firstPkgId
            }
        }
    } catch (e) {
        userPackages.value = []
        dnsProviders.value = []
    } finally {
        await reloadGroupSelect()
    }
}

const resetForm = () => {
  Object.assign(form, {
    id: 0,
    user_id: '',
    domains: '',
    origins: '',
    user_package_id: '',
    group_ids: [],
    group_id: '', 
    dns_provider_id: '',
    site_type: 'website',
    remark: ''
  })
  Object.assign(batchForm, {
      user_id: '',
      user_package_id: '',
      group_id: '',
      dns_provider_id: '',
      data: '',
      ignore_error: false,
      simpleDomains: '',
      simpleBackends: ''
  })
  currentBatchId.value = ''
  batchMode.value = 'simple'
  domainUsage.value = null
  domainLimitExceeded.value = false
}

const handleClosed = () => {
  singleFormRef.value?.resetFields()
}

const handleBatchValidation = (res) => {
    validBatchDomains.value = res.valid
}

const handleBatchComplete = () => {
    ElMessage.success('批量添加完成')
    emit('success') // Refresh list
}

const resetBatch = () => {
    currentBatchId.value = ''
    batchForm.simpleDomains = '' // Clear input
    // Keep user/package selection
}

const handleBatchClose = () => {
    visible.value = false
    emit('success')
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    if (domainLimitExceeded.value) {
        ElMessage.error(domainUsage.value?.message || '域名数量超过套餐限制')
        submitting.value = false
        return
    }
    if (activeTab.value === 'single') {
        await singleFormRef.value?.validate()
        const payload = { 
            ...form,
            user_id: Number(form.user_id),
            user_package_id: Number(form.user_package_id) || 0,
            dns_provider_id: Number(form.dns_provider_id) || 0,
            group_id: 0, 
            domains: form.domains.split('\n').map(s => s.trim()).filter(Boolean),
            backends: form.origins.split('\n').map(s => s.trim()).filter(Boolean),
            group_ids: Array.isArray(form.group_ids) ? form.group_ids : (form.group_ids ? [form.group_ids] : [])
        }
        delete payload.origins 
        
        if (form.id) {
          await request.put(`/sites/${form.id}`, payload)
        } else {
          await request.post('/sites', payload)
        }
        ElMessage.success('操作成功')
        emit('success')
        visible.value = false
    } else {
        // Construct Data if Simple Mode
        let dataStr = batchForm.data
        if (batchMode.value === 'simple') {
            if (!batchForm.simpleDomains || !batchForm.simpleBackends) {
                ElMessage.error('请填写域名和源站')
                submitting.value = false // Fix stuck loading
                return
            }
            const domains = validBatchDomains.value // Use validated list
            if (domains.length === 0) {
                 ElMessage.error('无有效域名')
                 submitting.value = false
                 return
            }
            // Normalize backends
            const backends = batchForm.simpleBackends.split('\n').map(s=>s.trim()).filter(Boolean).join(',')
            if (!backends) {
                 ElMessage.error('请填写源站')
                 submitting.value = false
                 return
            }
            
            // Build data string: domain=d1|ip=backends\ndomain=d2|ip=backends...
            // Note: backends comma separated for `ip=`? 
            // `parseBatchLine` usually parses `ip=a,b`. Yes.
            dataStr = domains.map(d => `domain=${d}|ip=${backends}`).join('\n')
        }

        if (!dataStr) {
            ElMessage.error('请输入网站数据')
            submitting.value = false
            return
        }
        
        const payload = {
            ...batchForm,
            data: dataStr,
            user_id: Number(batchForm.user_id),
            user_package_id: Number(batchForm.user_package_id) || 0,
            group_id: Number(batchForm.group_id) || 0,
            dns_provider_id: Number(batchForm.dns_provider_id) || 0,
        }
        
        const res = await request.post('/sites/batch', payload)
        
        ElMessage.success('批量添加任务已提交')
        visible.value = false
        emit('success')
    }
    
  } catch (err) {
      const msg = err?.response?.data?.error || err?.response?.data?.msg
      if (msg) {
        ElMessage.error(msg)
      }
      console.error(err)
  } finally {
    submitting.value = false
  }
}

function formatLimit(val) {
  const num = Number(val)
  if (!num || num <= 0) return '不限'
  return num
}

async function fetchDomainUsage() {
  const pkgId = Number(form.user_package_id) || 0
  if (!pkgId) {
    domainUsage.value = null
    domainLimitExceeded.value = false
    return
  }
  try {
    const params = { user_package_id: pkgId }
    if (props.isAdmin && form.user_id) {
      params.user_id = Number(form.user_id)
    }
    const res = await request.get('/domain_usage', { params })
    const data = res.data || res
    domainUsage.value = data
    domainLimitExceeded.value = !!data.exceeded
  } catch (e) {
    domainUsage.value = null
    domainLimitExceeded.value = false
  }
}
</script>

<style scoped>
.expand-more {
  text-align: center;
  margin: 10px 0;
  cursor: pointer;
  color: var(--el-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 13px;
}
.expand-more:hover { opacity: 0.8; }
.extra-fields {
  background: #f8f9fa;
  padding: 15px;
  border-radius: 4px;
  margin-bottom: 20px;
}
.limit-alert {
  margin: 8px 0 16px;
}
</style>
