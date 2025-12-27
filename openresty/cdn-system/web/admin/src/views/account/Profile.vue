<template>
  <div class="app-container">
    <el-card class="profile-card" shadow="never">
      <el-row :gutter="24" class="profile-row">
        <el-col :span="6">
          <div class="profile-item">
            <div class="label">用户名:</div>
            <div class="value">{{ profile.name }}</div>
          </div>
          <div class="profile-item">
            <div class="label">余额:</div>
            <div class="value">{{ balanceText }}</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="profile-item">
            <div class="label">QQ:</div>
            <div class="value">
              {{ profile.qq || '未设置' }}
              <el-button link type="primary" size="small" @click="editing.qq = true">编辑</el-button>
            </div>
          </div>
          <div v-if="editing.qq" class="inline-edit">
            <el-input v-model="profile.qq" placeholder="请输入QQ" size="small" style="width: 180px;" />
            <el-button type="primary" size="small" @click="saveProfile">保存</el-button>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="profile-item">
            <div class="label">密码:</div>
            <div class="value">
              ******
              <el-button link type="primary" size="small" @click="dialogVisible = true">修改</el-button>
            </div>
          </div>
        </el-col>
        <el-col :span="4">
          <div class="profile-item">
            <div class="label">注册时间:</div>
            <div class="value">{{ profile.create_at }}</div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">实名认证</div>
        <el-button type="primary" size="small" @click="saveCert">保存认证</el-button>
      </div>
      <el-tabs v-model="verifyTab">
        <el-tab-pane label="个人认证" name="personal">
          <el-form :model="personalForm" label-width="80px" class="section-form">
            <el-form-item label="姓名">
              <el-input v-model="personalForm.name" placeholder="请输入姓名" />
            </el-form-item>
            <el-form-item label="身份证号">
              <el-input v-model="personalForm.id" placeholder="请输入身份证号" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveCert">保存</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="企业认证" name="company">
          <el-form :model="companyForm" label-width="80px" class="section-form">
            <el-form-item label="身份证号">
              <el-input v-model="companyForm.name" placeholder="请输入企业名称" />
            </el-form-item>
            <el-form-item label="统一社会信用代码">
              <el-input v-model="companyForm.code" placeholder="请输入信用代码" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveCompanyCert">保存</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">手机绑定</div>
        <el-button type="primary" size="small" @click="savePhone">保存绑定</el-button>
      </div>
      <el-form :model="phoneForm" label-width="80px" class="section-form">
        <el-form-item label="手机号">
          <el-input v-model="phoneForm.mobile" placeholder="请输入手机号" />
        </el-form-item>
        <el-form-item label="验证码">
          <div class="code-row">
            <el-input v-model="phoneForm.code" placeholder="请输入验证码" />
            <el-button disabled>获取验证码</el-button>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="savePhone">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">邮箱绑定</div>
        <el-button type="primary" size="small" @click="saveEmail">保存绑定</el-button>
      </div>
      <el-form :model="emailForm" label-width="80px" class="section-form">
        <el-form-item label="姓名">
          <el-input v-model="emailForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="验证码">
          <div class="code-row">
            <el-input v-model="emailForm.code" placeholder="请输入验证码" />
            <el-button disabled>获取验证码</el-button>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveEmail">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">安全设置</div>
        <el-button type="primary" size="small" @click="saveSecurity">保存设置</el-button>
      </div>
      <el-form :model="securityForm" label-width="90px" class="section-form">
        <el-form-item label="IP白名单">
          <el-input v-model="securityForm.whitelist" placeholder="多个IP换行分隔" />
        </el-form-item>
        <el-form-item label="身份证号">
          <el-radio-group v-model="securityForm.verify">
            <el-radio value="none">不设置</el-radio>
            <el-radio value="sms">短信验证</el-radio>
            <el-radio value="email">邮箱验证</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveSecurity">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-dialog v-model="dialogVisible" title="修改密码" width="420px">
      <el-form :model="passwordForm" label-width="90px">
        <el-form-item label="身份证号">
          <el-input v-model="passwordForm.current" type="password" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="passwordForm.next" type="password" />
        </el-form-item>
        <el-form-item label="身份证号">
          <el-input v-model="passwordForm.confirm" type="password" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="savePassword">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const profile = reactive({
  name: '',
  balance: 0,
  qq: '',
  email: '',
  phone: '',
  create_at: '',
  cert_name: '',
  cert_no: '',
  cert_verified: false,
  white_ip: '',
  login_captcha: 'none'
})

const balanceText = computed(() => `${(profile.balance / 100).toFixed(2)}元`)

const editing = reactive({
  qq: false
})

const verifyTab = ref('personal')

const personalForm = reactive({
  name: '',
  id: ''
})

const companyForm = reactive({
  name: '',
  code: ''
})

const phoneForm = reactive({
  mobile: '',
  code: ''
})

const emailForm = reactive({
  email: '',
  code: ''
})

const securityForm = reactive({
  whitelist: '',
  verify: 'none'
})

const dialogVisible = ref(false)
const passwordForm = reactive({
  current: '',
  next: '',
  confirm: ''
})

const loadProfile = () => {
  request.get('/profile').then(res => {
    const data = res.data || {}
    profile.name = data.name || ''
    profile.balance = data.balance || 0
    profile.qq = data.qq || ''
    profile.email = data.email || ''
    profile.phone = data.phone || ''
    profile.create_at = data.create_at || ''
    profile.cert_name = data.cert_name || ''
    profile.cert_no = data.cert_no || ''
    profile.cert_verified = !!data.cert_verified
    profile.white_ip = data.white_ip || ''
    profile.login_captcha = data.login_captcha || 'none'

    personalForm.name = profile.cert_name
    personalForm.id = profile.cert_no
    companyForm.name = profile.cert_name
    companyForm.code = profile.cert_no
    phoneForm.mobile = profile.phone
    emailForm.email = profile.email
    securityForm.whitelist = profile.white_ip
    securityForm.verify = profile.login_captcha || 'none'
  })
}

const updateProfile = payload => {
  return request.put('/profile', payload).then(() => {
    ElMessage.success('\u4fdd\u5b58\u6210\u529f')
  })
}

const saveProfile = () => {
  editing.qq = false
  updateProfile({
    qq: profile.qq,
    email: profile.email,
    phone: profile.phone,
    white_ip: profile.white_ip,
    login_captcha: profile.login_captcha,
    cert_name: profile.cert_name,
    cert_no: profile.cert_no
  })
}

const saveCert = () => {
  profile.cert_name = personalForm.name
  profile.cert_no = personalForm.id
  saveProfile()
}

const saveCompanyCert = () => {
  profile.cert_name = companyForm.name
  profile.cert_no = companyForm.code
  saveProfile()
}

const savePhone = () => {
  profile.phone = phoneForm.mobile
  saveProfile()
}

const saveEmail = () => {
  profile.email = emailForm.email
  saveProfile()
}

const saveSecurity = () => {
  profile.white_ip = securityForm.whitelist
  profile.login_captcha = securityForm.verify
  saveProfile()
}

const savePassword = () => {
  if (!passwordForm.current || !passwordForm.next) {
    ElMessage.warning('\u8bf7\u8f93\u5165\u5b8c\u6574\u4fe1\u606f')
    return
  }
  if (passwordForm.next !== passwordForm.confirm) {
    ElMessage.warning('\u4e24\u6b21\u5bc6\u7801\u4e0d\u4e00\u81f4')
    return
  }
  request.put('/password', {
    current: passwordForm.current,
    next: passwordForm.next
  }).then(() => {
    dialogVisible.value = false
    passwordForm.current = ''
    passwordForm.next = ''
    passwordForm.confirm = ''
    ElMessage.success('\u4fdd\u5b58\u6210\u529f')
  })
}

onMounted(() => loadProfile())
</script>

<style scoped>
.profile-card {
  margin-bottom: 16px;
}
.profile-row {
  align-items: center;
}
.profile-item {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.label {
  color: #606266;
}
.value {
  color: #303133;
}
.inline-edit {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
}
.section-card {
  margin-bottom: 16px;
}
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.section-title {
  font-weight: 600;
  color: #303133;
}
.section-form {
  max-width: 520px;
}
.code-row {
  display: flex;
  gap: 10px;
}
</style>
