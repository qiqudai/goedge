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
              <el-radio value="zerossl">ZeroSSL（推荐）</el-radio>
              <el-radio value="letsencrypt">Let's Encrypt</el-radio>
              <el-radio value="buypass">BuyPass</el-radio>
              <el-radio value="google">Google CA</el-radio>
            </el-radio-group>
          </el-form-item>

          <template v-if="showCertFields">
            <el-form-item label="证书">
              <el-input v-model="form.cert" type="textarea" :rows="4" placeholder="-----BEGIN CERTIFICATE-----" @blur="handleCertPemBlur" />
            </el-form-item>
            <el-form-item label="密钥">
              <el-input v-model="form.key" type="textarea" :rows="4" placeholder="-----BEGIN PRIVATE KEY-----" @blur="handleKeyPemBlur" />
            </el-form-item>
          </template>

          <el-form-item label="域名" v-if="form.type !== 'upload'">
             <el-input v-model="form.domain" placeholder="输入域名, 多个域名空格分隔" />
          </el-form-item>

          <el-form-item label="DNS 接口" v-if="form.type !== 'upload'">
             <el-select v-model="form.dnsapi" clearable placeholder="不选择 (HTTP验证)" style="width: 100%;">
                <el-option label="不选择 (HTTP验证)" :value="0" />
                <el-option v-for="d in dnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
             </el-select>
             <div class="form-helper" v-if="!form.dnsapi">
               不选择 DNS 接口时，需要将域名解析 CNAME 地址，这种方式无法申请通配符域名；
             </div>
             <div class="form-helper" v-else>
               选择 DNS 接口时，可以申请所有类型的域名，包括通配符，申请成功率比较高。
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
              <el-radio value="zerossl">ZeroSSL（推荐）</el-radio>
              <el-radio value="letsencrypt">Let's Encrypt</el-radio>
              <el-radio value="buypass">BuyPass</el-radio>
              <el-radio value="google">Google CA</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="域名">
            <el-input v-model="batchForm.domains" type="textarea" :rows="6" placeholder="一行一个域名" />
          </el-form-item>
          <el-form-item label="DNS 接口">
             <el-select v-model="batchForm.dnsapi" clearable placeholder="不选择 (HTTP验证)" style="width: 100%;">
                <el-option label="不选择 (HTTP验证)" :value="0" />
                <el-option v-for="d in dnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
             </el-select>
             <div class="form-helper">
               这里选择的 DNS 接口将应用于所有批量申请的域名。
             </div>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="泛证书申请" name="wildcard" v-if="!certId">
        <el-form :model="wildcardForm" label-width="100px">
          <el-form-item label="用户" v-if="isAdmin">
            <el-select
              v-model="wildcardForm.user_id"
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
            <el-radio-group v-model="wildcardForm.type">
              <el-radio value="zerossl">ZeroSSL（推荐）</el-radio>
              <el-radio value="letsencrypt">Let's Encrypt</el-radio>
              <el-radio value="buypass">BuyPass</el-radio>
              <el-radio value="google">Google CA</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="验证域名">
            <el-input v-model="wildcardForm.domain" placeholder="*.example.com" />
          </el-form-item>
          <el-form-item label="DNS 接口">
            <el-select v-model="wildcardForm.dnsapi" clearable placeholder="不选择 (手动TXT)" style="width: 100%;">
              <el-option label="不选择 (手动TXT)" :value="0" />
              <el-option v-for="d in dnsapiOptions" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
            <div class="form-helper" v-if="!wildcardForm.dnsapi">
              不选择 DNS 接口时，需要手动添加 TXT 记录完成验证。
            </div>
            <div class="form-helper" v-else>
              选择 DNS 接口时会自动设置 TXT 记录，无需手动解析。
            </div>
          </el-form-item>
        </el-form>

        <div v-if="!wildcardForm.dnsapi" class="dns-manual">
          <div class="dns-title">请按以下列表做TXT解析:</div>
          <div class="dns-line">验证域名：{{ wildcardForm.domain || '-' }}</div>
          <el-table
            v-if="wildcardChallenge"
            :data="[wildcardChallenge]"
            border
            size="small"
            style="width: 100%"
          >
            <el-table-column label="解析域名">
              <template #default="{ row }">
                <span>{{ row.fqdn }}</span>
                <el-icon class="copy-icon" @click="copyText(row.record_name)" v-if="row.record_name">
                  <CopyDocument />
                </el-icon>
              </template>
            </el-table-column>
            <el-table-column label="记录值">
              <template #default="{ row }">
                <span>{{ row.record_value }}</span>
                <el-icon class="copy-icon" @click="copyText(row.record_value)" v-if="row.record_value">
                  <CopyDocument />
                </el-icon>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="100">
              <template #default="{ row }">
                <span>{{ row.record_type || 'TXT' }}</span>
              </template>
            </el-table-column>
          </el-table>
          <div class="form-helper" v-else>提交申请后会生成TXT记录信息。</div>
          <div class="dns-remark">
            备注：<br />
            解析域名需要一定时间来生效,完成所以上所有解析操作后,请等待1分钟后再点击验证按钮<br />
            可通过CMD命令来手动验证域名解析是否生效: nslookup -q=txt _acme-challenge.域名<br />
            若您使用的是宝塔云解析插件,阿里云DNS,DnsPod作为DNS,可使用DNS接口自动解析
          </div>
          <el-button type="primary" size="small" @click="verifyDNSChallenge" :disabled="!wildcardCertId">验证</el-button>
        </div>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose" :disabled="submitting">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">确定</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, computed } from 'vue'
import { InfoFilled, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import { looksLikeCertificatePem, looksLikePrivateKeyPem, normalizeUploadPemFields } from '@/utils/pem'

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
const originalType = ref('upload')
const hasStoredCert = ref(false)
const showCertFields = computed(() => form.type === 'upload' || (props.certId && hasStoredCert.value && form.type === originalType.value))

// Data Sources
const userOptions = ref([])
const userLoading = ref(false)
const submitting = ref(false)
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

const wildcardForm = reactive({
  user_id: null,
  type: 'zerossl',
  domain: '',
  dnsapi: 0,
  auto_renew: true
})

const wildcardCertId = ref(0)
const wildcardChallenge = ref(null)

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

watch(() => wildcardForm.domain, () => {
  wildcardCertId.value = 0
  wildcardChallenge.value = null
})

watch(() => wildcardForm.dnsapi, (val) => {
  if (val) {
    wildcardChallenge.value = null
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
   originalType.value = form.type
   form.domain = data.domain 
   form.dnsapi = data.dnsapi || 0
   form.cert = data.cert 
   form.key = data.key 
   if (form.type === 'upload' && form.key && looksLikeCertificatePem(form.key) && !looksLikePrivateKeyPem(form.key)) {
     form.key = ''
     ElMessage.warning('密钥数据无效（存的是证书内容），请重新粘贴私钥')
   }
   form.auto_renew = data.auto_renew !== false
   hasStoredCert.value = !!(data.cert || data.key)
   
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
  originalType.value = form.type
  form.domain = ''
  form.dnsapi = 0
  form.cert = ''
  form.key = ''
  form.auto_renew = true
  hasStoredCert.value = false
  
  batchForm.user_id = null
  batchForm.type = 'zerossl'
  batchForm.domains = ''
  batchForm.dnsapi = 0
  batchForm.auto_renew = true

  wildcardForm.user_id = null
  wildcardForm.type = 'zerossl'
  wildcardForm.domain = ''
  wildcardForm.dnsapi = 0
  wildcardForm.auto_renew = true
  wildcardCertId.value = 0
  wildcardChallenge.value = null
}

const handleCertPemBlur = () => {
  if (form.type !== 'upload') return
  const normalized = normalizeUploadPemFields(form.cert, form.key)
  form.cert = normalized.cert
  if (normalized.key) {
    form.key = normalized.key
  }
}

const handleKeyPemBlur = () => {
  if (form.type !== 'upload') return
  const normalized = normalizeUploadPemFields(form.cert, form.key)
  if (normalized.key) {
    form.key = normalized.key
  }
  if (!form.cert && normalized.cert) {
    form.cert = normalized.cert
  }
  if (form.key && looksLikeCertificatePem(form.key) && !looksLikePrivateKeyPem(form.key)) {
    ElMessage.warning('密钥栏不能填写证书内容，请粘贴 BEGIN PRIVATE KEY')
    form.key = ''
  }
}

const submit = async () => {
  if (submitting.value) return
  if (activeTab.value === 'batch') {
     await submitBatch()
     return
  }
  if (activeTab.value === 'wildcard') {
     await submitWildcard()
     return
  }

  // Submit Single
  // Logic from Certs.vue:
  // If not upload and new, maybe use batch logic if multiple domains?
  // But let's stick to standard single update/create for now unless domains > 1 and it's new.

  const payload = { ...form }
  if (showCertFields.value) {
    const normalized = normalizeUploadPemFields(payload.cert, payload.key)
    payload.cert = normalized.cert
    payload.key = normalized.key
    if (payload.key && looksLikeCertificatePem(payload.key) && !looksLikePrivateKeyPem(payload.key)) {
      ElMessage.warning('密钥栏不能填写证书内容，请粘贴私钥')
      return
    }
  }
  if (!showCertFields.value) {
    delete payload.cert
    delete payload.key
  }
  // Backend permissions check:
  // If not admin, user_id should be ignored by backend or enforced to current user.
  // Frontend sends it if isAdmin is true.

  submitting.value = true
  try {
    if (props.certId) {
      await request.put(`/certs/${props.certId}`, payload)
      ElMessage.success('更新成功')
      handleClose()
      emits('saved')
    } else {
      // New
      // Handle split domains if not upload
      if (form.type !== 'upload') {
          const domains = form.domain.split(/[\s,;]+/).filter(Boolean)
          const hasWildcard = domains.some(d => d.trim().startsWith('*.'))
          if (hasWildcard && !form.dnsapi) {
              ElMessage.warning('泛证书请在泛证书申请页或选择 DNS 接口')
              return
          }
          const batchPayload = {
              user_id: Number(form.user_id) || 0,
              type: form.type,
              domains: domains,
              dnsapi: Number(form.dnsapi) || 0,
              auto_renew: true
          }
          await request.post('/certs/batch', batchPayload)
          ElMessage.success('已提交申请')
          handleClose()
          emits('saved')
      } else {
          await request.post('/certs', payload)
          ElMessage.success('添加成功')
          handleClose()
          emits('saved')
      }
    }
  } finally {
    submitting.value = false
  }
}

const submitBatch = async () => {
    if (submitting.value) return
    const domains = batchForm.domains.split('\n').map(s=>s.trim()).filter(Boolean)
    if (!domains.length) {
        ElMessage.warning('请输入域名')
        return
    }
    const hasWildcard = domains.some(d => d.startsWith('*.'))
    if (hasWildcard && !batchForm.dnsapi) {
        ElMessage.warning('泛证书必须选择 DNS 接口')
        return
    }
    const payload = {
        user_id: Number(batchForm.user_id) || 0,
        type: batchForm.type,
        domains: domains,
        dnsapi: Number(batchForm.dnsapi) || 0,
        auto_renew: true
    }
    submitting.value = true
    try {
        await request.post('/certs/batch', payload)
        ElMessage.success('批量提交成功')
        handleClose()
        emits('saved')
    } finally {
        submitting.value = false
    }
}

const submitWildcard = async () => {
  if (submitting.value) return
  const domain = (wildcardForm.domain || '').trim()
  if (!domain) {
    ElMessage.warning('请输入泛域名')
    return
  }
  if (!domain.startsWith('*.')) {
    ElMessage.warning('泛证书域名需以 *.' + ' 开头')
    return
  }

  const payload = {
    user_id: Number(wildcardForm.user_id) || 0,
    type: wildcardForm.type,
    domain: domain,
    dnsapi: Number(wildcardForm.dnsapi) || 0,
    auto_renew: true
  }

  submitting.value = true
  try {
    const res = await request.post('/certs/wildcard', payload)
    wildcardCertId.value = res.id || res.data?.id || 0
    ElMessage.success('已提交申请')
    if (!wildcardForm.dnsapi && wildcardCertId.value) {
      await loadWildcardChallenge(wildcardCertId.value)
    }
    emits('saved')
  } catch (e) {
    // errors are handled by request interceptor
  } finally {
    submitting.value = false
  }
}

const loadWildcardChallenge = async (certId) => {
  wildcardChallenge.value = null
  if (!certId) return
  for (let i = 0; i < 5; i++) {
    try {
      const res = await request.get(`/certs/${certId}/dns_challenge`)
      const data = res.data || res
      if (data) {
        wildcardChallenge.value = data
        return
      }
      await new Promise(resolve => setTimeout(resolve, 1000))
    } catch (e) {
      return
    }
  }
}

const verifyDNSChallenge = async () => {
  if (!wildcardCertId.value) {
    ElMessage.warning('请先提交申请')
    return
  }
  try {
    await request.post(`/certs/${wildcardCertId.value}/verify_dns`)
    ElMessage.success('验证成功，请等待签发完成')
  } catch (e) {
    // handled globally
  }
}

const copyText = async (text) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

</script>

<style scoped>
.form-helper {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 5px;
}

.dns-manual {
  margin-top: 10px;
}

.dns-title {
  font-weight: 600;
  margin-bottom: 6px;
}

.dns-line {
  margin-bottom: 8px;
}

.dns-remark {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.copy-icon {
  margin-left: 6px;
  cursor: pointer;
  color: #409eff;
}
</style>
