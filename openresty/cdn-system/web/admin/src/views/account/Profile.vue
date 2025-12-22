<template>
  <div class="app-container">
    <el-card class="profile-card" shadow="never">
      <el-row :gutter="24" class="profile-row">
        <el-col :span="6">
          <div class="profile-item">
            <div class="label">用户名:</div>
            <div class="value">{{ profile.username }}</div>
          </div>
          <div class="profile-item">
            <div class="label">余额:</div>
            <div class="value">{{ profile.balance }}</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="profile-item">
            <div class="label">QQ:</div>
            <div class="value">
              {{ profile.qq || '未设置' }}
              <el-button link type="primary" size="small" @click="editing.qq = true">修改</el-button>
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
            <div class="value">{{ profile.registered_at }}</div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">实名认证</div>
        <el-button type="primary" size="small">立即认证</el-button>
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
              <el-button type="primary">确定</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="企业认证" name="company">
          <el-form :model="companyForm" label-width="80px" class="section-form">
            <el-form-item label="企业名称">
              <el-input v-model="companyForm.name" placeholder="请输入企业名称" />
            </el-form-item>
            <el-form-item label="社会信用代码">
              <el-input v-model="companyForm.code" placeholder="请输入社会信用代码" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary">确定</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">手机绑定</div>
        <el-button type="primary" size="small">立即绑定</el-button>
      </div>
      <el-form :model="phoneForm" label-width="80px" class="section-form">
        <el-form-item label="手机号">
          <el-input v-model="phoneForm.mobile" placeholder="请输入手机号" />
        </el-form-item>
        <el-form-item label="验证码">
          <div class="code-row">
            <el-input v-model="phoneForm.code" placeholder="短信验证码" />
            <el-button>获取验证码</el-button>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary">确定</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">邮箱绑定</div>
        <el-button type="primary" size="small">立即绑定</el-button>
      </div>
      <el-form :model="emailForm" label-width="80px" class="section-form">
        <el-form-item label="邮箱">
          <el-input v-model="emailForm.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="验证码">
          <div class="code-row">
            <el-input v-model="emailForm.code" placeholder="邮箱验证码" />
            <el-button>获取验证码</el-button>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary">确定</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="section-card" shadow="never">
      <div class="section-header">
        <div class="section-title">登录安全设置</div>
        <el-button type="primary" size="small">立即设置</el-button>
      </div>
      <el-form :model="securityForm" label-width="90px" class="section-form">
        <el-form-item label="登录白名单">
          <el-input v-model="securityForm.whitelist" placeholder="多个IP空格分隔" />
        </el-form-item>
        <el-form-item label="登录验证">
          <el-radio-group v-model="securityForm.verify">
            <el-radio label="none">无</el-radio>
            <el-radio label="sms">短信验证码</el-radio>
            <el-radio label="email">邮件验证码</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item>
          <el-button type="primary">确定</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-dialog v-model="dialogVisible" title="修改密码" width="420px">
      <el-form :model="passwordForm" label-width="90px">
        <el-form-item label="当前密码">
          <el-input v-model="passwordForm.current" type="password" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="passwordForm.next" type="password" />
        </el-form-item>
        <el-form-item label="确认密码">
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
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'

const profile = reactive({
  username: 'feiyang666',
  balance: '0元',
  qq: '',
  registered_at: '2025-11-06 16:26:17'
})

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
  email: 'feiyang666@cdn.cn',
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

const saveProfile = () => {
  editing.qq = false
  ElMessage.success('资料已保存')
}

const savePassword = () => {
  dialogVisible.value = false
  ElMessage.success('密码已更新')
}
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
