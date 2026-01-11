<template>
  <el-dialog
    :model-value="visible"
    title="编辑用户"
    width="700px"
    @close="handleClose"
    :close-on-click-modal="false"
  >
    <el-tabs v-model="activeTab" type="card">
      <!-- Tab 1: Basic Info -->
      <el-tab-pane label="基础信息" name="basic">
        <el-form :model="form" label-width="100px" ref="basicForm">
          <el-form-item label="ID">
             <span>{{ form.id }}</span>
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input v-model="form.email" placeholder="请输入邮箱" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="form.name" placeholder="请输入用户名" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="form.des" type="textarea" :rows="2" placeholder="请输入备注" />
          </el-form-item>
          <el-form-item label="手机号">
            <el-input v-model="form.phone" placeholder="请输入手机号" />
          </el-form-item>
          <el-form-item label="QQ">
            <el-input v-model="form.qq" placeholder="请输入QQ" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" type="password" placeholder="留空则不修改" show-password />
          </el-form-item>
          <el-form-item label="用户分组">
             <el-select v-model="form.group_id" placeholder="请选择">
                <!-- Group options usually loaded from API, assume props or fetch -->
                <el-option label="默认分组" :value="0" />
             </el-select>
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="form.enable" />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- Tab 2: Real-name Info -->
      <el-tab-pane label="实名信息" name="realname">
        <el-form :model="form" label-width="120px">
          <el-form-item label="姓名">
            <el-input v-model="form.cert_name" placeholder="请输入姓名" />
          </el-form-item>
          <el-form-item label="身份证">
            <el-input v-model="form.cert_no" placeholder="请输入身份证" />
          </el-form-item>
          <el-form-item label="公司名称">
            <el-input v-model="form.company" placeholder="请输入公司名称" />
          </el-form-item>
          <el-form-item label="社会信用代码">
            <el-input v-model="form.tea_code" placeholder="请输入社会信用代码" />
          </el-form-item>
          <el-form-item label="认证状态">
             <div class="auth-status-row">
                 <span>个人实名: <el-tag :type="form.cert_name ? 'success' : 'info'">{{ form.cert_name ? '已认证' : '未认证' }}</el-tag></span>
                 <span style="margin-left: 20px;">企业实名: <el-tag :type="form.company ? 'success' : 'info'">{{ form.company ? '已认证' : '未认证' }}</el-tag></span>
             </div>
          </el-form-item>
          
          <el-divider content-position="left">二次实名</el-divider>
          
          <el-form-item label="开关">
            <el-switch v-model="form.secondary_auth" />
            <div class="help-text">开启后，用户必须使用首次实名认证的身份证进行二次实名认证；<br>将自动向用户发送邮件和手机短信通知</div>
          </el-form-item>
          <el-form-item label="截止时间">
             <el-date-picker
                v-model="form.secondary_auth_deadline"
                type="datetime"
                placeholder="如2026-01-01 12:00:00,留空则不限制"
                style="width: 100%"
                value-format="YYYY-MM-DD HH:mm:ss"
             />
          </el-form-item>
          <el-form-item label="认证过期">
             <el-radio-group v-model="form.secondary_auth_action">
                <el-radio value="">不处理</el-radio>
                <el-radio value="lock">锁定网站</el-radio>
             </el-radio-group>
          </el-form-item>
          <el-form-item label="认证状态">
             <el-tag type="info">未开始</el-tag> <!-- Placeholder logic -->
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- Tab 3: Security -->
      <el-tab-pane label="登录安全" name="security">
        <el-form :model="form" label-width="100px">
           <el-form-item label="登录验证码">
             <el-radio-group v-model="form.login_captcha">
                <el-radio value="none">无</el-radio>
                <el-radio value="sms">短信</el-radio>
                <el-radio value="email">邮件</el-radio>
             </el-radio-group>
           </el-form-item>
           <el-form-item label="登录白名单">
              <el-input v-model="form.white_ip" type="textarea" :rows="4" placeholder="多个IP空格分隔" />
           </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="submit" :loading="saving">保存</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'

const props = defineProps({
  visible: Boolean,
  userData: {
    type: Object,
    default: () => ({})
  }
})

const emits = defineEmits(['update:visible', 'saved'])

const activeTab = ref('basic')
const saving = ref(false)

const form = reactive({
  id: 0,
  email: '',
  name: '',
  des: '',
  phone: '',
  qq: '',
  password: '',
  group_id: 0,
  enable: true,
  
  cert_name: '',
  cert_no: '',
  company: '',
  tea_code: '',
  
  secondary_auth: false,
  secondary_auth_deadline: '',
  secondary_auth_action: '',
  
  login_captcha: 'none',
  white_ip: ''
})

watch(() => props.visible, (val) => {
  if (val) {
    activeTab.value = 'basic'
    initForm(props.userData)
  }
})

const initForm = (data) => {
  form.id = data.id || 0
  form.email = data.email || ''
  form.name = data.name || '' // Username
  form.des = data.des || ''
  form.phone = data.phone || ''
  form.qq = data.qq || ''
  form.password = ''
  form.group_id = data.group_id || 0
  form.enable = data.enable !== false
  
  form.cert_name = data.cert_name || ''
  form.cert_no = data.cert_no || ''
  form.company = data.company || ''
  form.tea_code = data.tea_code || ''
  
  form.secondary_auth = !!data.secondary_auth
  form.secondary_auth_deadline = data.secondary_auth_deadline || ''
  form.secondary_auth_action = data.secondary_auth_action || ''
  
  form.login_captcha = data.login_captcha || 'none'
  form.white_ip = data.white_ip || ''
}

const handleClose = () => {
  emits('update:visible', false)
}

const submit = () => {
  saving.value = true
  const payload = { ...form }
  
  request.put(`/users/${form.id}`, payload).then(() => {
    ElMessage.success('保存成功')
    handleClose()
    emits('saved')
  }).finally(() => {
    saving.value = false
  })
}
</script>

<style scoped>
.help-text {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 5px;
}
.auth-status-row {
  display: flex; 
  align-items: center;
}
</style>
