<template>
  <el-dialog
    :model-value="visible"
    :title="title"
    width="640px"
    @close="handleClose"
    :close-on-click-modal="false"
  >
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane label="单个证书" name="single">
        <el-form :model="form" label-width="100px" ref="formRef">
          <!-- 用户 (管理员可见) -->
          <el-form-item label="用户" v-if="isAdmin">
            <template #label>
              <span>用户</span>
              <el-tooltip content="只有管理员账号显示该列 并有权限修改" placement="top">
                <el-icon><InfoFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-select
              v-model="form.user_id"
              filterable
              remote
              clearable
              placeholder="搜索用户ID或账号"
              :remote-method="searchUsers"
              :loading="userLoading"
              style="width: 100%"
            >
              <el-option
                v-for="u in userOptions"
                :key="u.id"
                :label="formatUserLabel(u)"
                :value="u.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="名称">
            <el-input v-model="form.name" placeholder="输入证书名称" />
          </el-form-item>

          <el-form-item label="备注">
            <el-input v-model="form.des" type="textarea" :rows="2" placeholder="备注" />
          </el-form-item>

          <el-form-item label="类型">
            <el-radio-group v-model="form.type">
              <el-radio value="upload">自己上传</el-radio>
              <el-radio value="zerossl">ZeroSSL(推荐)</el-radio>
              <el-radio value="letsencrypt">Let's Encrypt</el-radio>
              <el-radio value="buypass">BuyPass</el-radio>
              <el-radio value="google">GoogleCA</el-radio>
            </el-radio-group>
          </el-form-item>

          <template v-if="form.type === 'upload'">
            <el-form-item label="证书">
              <el-input v-model="form.cert" type="textarea" :rows="4" placeholder="-----BEGIN CERTIFICATE-----" />
            </el-form-item>
            <el-form-item label="密钥">
              <el-input v-model="form.key" type="textarea" :rows="4" placeholder="-----BEGIN PRIVATE KEY-----" />
            </el-form-item>
          </template>

          <el-form-item label="域名" v-if="form.type !== 'upload'">
             <el-input v-model="form.domain" placeholder="输入域名, 多个域名空格分隔" />
          </el-form-item>

          <el-form-item label="DNS API" v-if="form.type !== 'upload'">
             <el-select v-model="form.dnsapi" clearable placeholder="不选择 (HTTP验证)" style="width: 100%;">
                <el-option label="不选择 (HTTP验证)" :value="0" />
                <el-option v-for="d in dnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
             </el-select>
             <div class="form-helper" v-if="!form.dnsapi">
               不选择DNS API时，需要将域名解析CNAME地址，这种方式无法申请通配符域名；
             </div>
             <div class="form-helper" v-else>
               选择DNS API时，可以申请所有类型的域名，包括通配符，申请成功率比较高。
             </div>
          </el-form-item>

        </el-form>
      </el-tab-pane>

      <!-- 批量申请 (Create Only) -->
      <el-tab-pane label="批量申请" name="batch" v-if="!certId">
        <el-form :model="batchForm" label-width="100px">
           <el-form-item label="用户" v-if="isAdmin">
            <el-select
              v-model="batchForm.user_id"
              filterable
              remote
              clearable
              placeholder="搜索用户ID或账号"
              :remote-method="searchUsers"
              :loading="userLoading"
              style="width: 100%"
            >
              <el-option
                v-for="u in userOptions"
                :key="u.id"
                :label="formatUserLabel(u)"
                :value="u.id"
              />
            </el-select>
          </el-form-item>
           <el-form-item label="类型">
            <el-radio-group v-model="batchForm.type">
              <el-radio value="zerossl">ZeroSSL(推荐)</el-radio>
              <el-radio value="letsencrypt">Let's Encrypt</el-radio>
              <el-radio value="buypass">BuyPass</el-radio>
              <el-radio value="google">GoogleCA</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="域名">
            <el-input v-model="batchForm.domains" type="textarea" :rows="6" placeholder="一行一个域名" />
          </el-form-item>
          <el-form-item label="DNS API">
             <el-select v-model="batchForm.dnsapi" clearable placeholder="不选择 (HTTP验证)" style="width: 100%;">
                <el-option label="不选择 (HTTP验证)" :value="0" />
                <el-option v-for="d in dnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
             </el-select>
             <div class="form-helper">
               这里选择的DNS API将应用于所有批量申请的域名。
             </div>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="submit">确定</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, computed, onMounted } from 'vue'
import { InfoFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const props = defineProps({
  visible: Boolean,
  certId: {
    type: Number,
    default: 0
  },
  isAdmin: {
    type: Boolean,
    default: false
  },
  initialData: {
    type: Object,
    default: () => ({})
  }
})

const emits = defineEmits(['update:visible', 'saved', 'close'])

const activeTab = ref('single')
const title = computed(() => props.certId ? '编辑证书' : '添加证书')

// Data Sources
const userOptions = ref([])
const userLoading = ref(false)
const dnsapiOptions = ref([])

// Forms
const form = reactive({
  id: 0,
  user_id: null,
  name: '',
  des: '',
  type: 'upload',
  domain: '',
  dnsapi: 0,
  cert: '',
  key: '',
  auto_renew: true
})

const batchForm = reactive({
  user_id: null,
  type: 'zerossl',
  domains: '',
  dnsapi: 0,
  auto_renew: true
})

// Watchers
watch(() => props.visible, (val) => {
  if (val) {
    loadDnsapi()
    if (props.certId) {
      activeTab.value = 'single'
      initForm(props.initialData)
    } else {
      activeTab.value = 'single' // or default to batch? default single.
      resetForm()
    }
  }
})

// Validation Helper
const formRef = ref(null)

// Methods
const handleClose = () => {
  emits('update:visible', false)
  emits('close')
}

const formatUserLabel = (u) => {
  if (!u) return ''
  return u.name ? `${u.name} (id: ${u.id})` : `id: ${u.id}`
}

const searchUsers = (query) => {
  if (query) {
    userLoading.value = true
    request.get('/users', { params: { keyword: query, pageSize: 20 } }).then(res => {
        userOptions.value = res.data?.list || res.list || []
        userLoading.value = false
    }).catch(() => {
        userLoading.value = false
    })
  }
}

const loadDnsapi = () => {
  request.get('/dnsapi').then(res => {
    dnsapiOptions.value = res.data?.list || res.list || []
  })
}

const initForm = (data) => {
   form.id = data.id 
   form.user_id = data.uid || data.user_id // Handle mapping
   form.name = data.name 
   form.des = data.des || data.description || '' // Handle key mismatch
   form.type = data.type || 'upload'
   form.domain = data.domain 
   form.dnsapi = data.dnsapi || 0
   form.cert = data.cert 
   form.key = data.key 
   form.auto_renew = data.auto_renew !== false
   
   // Pre-fill user options if editing and we have user info
   if (form.user_id) {
       // Ideally we should add the current user to options so it shows up
       if (data.user_name || data.userName) {
           userOptions.value = [{ id: form.user_id, name: data.user_name || data.userName }]
       } else {
         // Try to fetch if name not present? Or just show ID
         if (!userOptions.value.find(u => u.id === form.user_id)) {
            userOptions.value.push({ id: form.user_id, name: 'Loading...' }) 
         }
       }
   }
}

const resetForm = () => {
  form.id = 0
  form.user_id = null
  form.name = ''
  form.des = ''
  form.type = 'upload'
  form.domain = ''
  form.dnsapi = 0
  form.cert = ''
  form.key = ''
  form.auto_renew = true
  
  batchForm.user_id = null
  batchForm.type = 'zerossl'
  batchForm.domains = ''
  batchForm.dnsapi = 0
}

const submit = () => {
  if (activeTab.value === 'batch') {
     submitBatch()
     return
  }

  // Submit Single
  // Logic from Certs.vue:
  // If not upload and new, maybe use batch logic if multiple domains?
  // But let's stick to standard single update/create for now unless domains > 1 and it's new.
  
  const payload = { ...form }
  // Backend permissions check:
  // If not admin, user_id should be ignored by backend or enforced to current user.
  // Frontend sends it if isAdmin is true.
  
  if (props.certId) {
    request.put(`/certs/${props.certId}`, payload).then(() => {
      ElMessage.success('更新成功')
      handleClose()
      emits('saved')
    })
  } else {
    // New
    // Handle split domains if not upload
    if (form.type !== 'upload') {
        const domains = form.domain.split(/[\s,;]+/).filter(Boolean)
        const batchPayload = {
            user_id: Number(form.user_id) || 0,
            type: form.type,
            domains: domains,
            dnsapi: Number(form.dnsapi) || 0,
            auto_renew: true
        }
        request.post('/certs/batch', batchPayload).then(() => {
             ElMessage.success('已提交申请')
             handleClose()
             emits('saved')
        })
    } else {
        request.post('/certs', payload).then(() => {
           ElMessage.success('添加成功')
           handleClose()
           emits('saved')
        })
    }
  }
}

const submitBatch = () => {
    const domains = batchForm.domains.split('\n').map(s=>s.trim()).filter(Boolean)
    if (!domains.length) {
        ElMessage.warning('请输入域名')
        return
    }
    const payload = {
        user_id: Number(batchForm.user_id) || 0,
        type: batchForm.type,
        domains: domains,
        dnsapi: Number(batchForm.dnsapi) || 0,
        auto_renew: true
    }
    request.post('/certs/batch', payload).then(() => {
         ElMessage.success('批量提交成功')
         handleClose()
         emits('saved')
    })
}

</script>

<style scoped>
.form-helper {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 5px;
}
</style>
