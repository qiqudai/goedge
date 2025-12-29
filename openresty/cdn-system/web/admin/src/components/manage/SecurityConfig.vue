<template>
  <div class="security-config">
    <div class="section-title">CC 防护</div>
    <el-form label-width="120px" class="config-form">
      <el-form-item label="默认规则" style="width: 720px">
        <el-radio-group v-model="securitySettings.cc.mode">
          <el-radio :value="10002">关闭</el-radio>
          <el-radio :value="10003">宽松</el-radio>
          <el-radio :value="10004">普通</el-radio>
          <el-radio :value="10005">严格</el-radio>
          <el-radio :value="10006">JS验证</el-radio>
          <el-radio :value="10008">验证码</el-radio>
          <el-radio :value="10009">自定义</el-radio>
        </el-radio-group>
        <div class="form-helper">不同模式对应不同的防御级别</div>
      </el-form-item>
      
      <el-form-item label="自动切换">
        <div style="display: flex; align-items: center; gap: 10px;">
          <el-switch v-model="securitySettings.cc.autoSwitch.enable" />
          <span v-if="securitySettings.cc.autoSwitch.enable" style="font-size: 13px;">
            当QPS超过 
            <el-input v-model="securitySettings.cc.autoSwitch.qps" size="small" style="width: 80px" /> 时，
            自动切换到 
            <el-select v-model="securitySettings.cc.autoSwitch.rule" size="small" style="width: 100px;">
              <el-option label="关闭" value="close" />
              <el-option label="宽松" value="lenient" />
              <el-option label="普通" value="normal" />
              <el-option label="严格" value="strict" />
              <el-option label="JS验证" value="js" />
              <el-option label="验证码" value="captcha" />
            </el-select>
          </span>
        </div>
      </el-form-item>
      
      <div class="divider"></div>
      
      <div class="section-title">黑白名单</div>
      <el-form-item label="IP黑名单">
        <el-input type="textarea" v-model="securitySettings.ip.black" :rows="3" placeholder="一行一个IP" />
      </el-form-item>
      <el-form-item label="IP白名单">
        <el-input type="textarea" v-model="securitySettings.ip.white" :rows="3" placeholder="一行一个IP" />
      </el-form-item>

      <div class="divider"></div>

      <div class="section-title">UA黑白名单</div>
      <el-form-item label="UA黑名单">
        <el-input type="textarea" v-model="securitySettings.ua.black" :rows="3" placeholder="一行一个UA keyword" />
      </el-form-item>
      <el-form-item label="UA白名单">
        <el-input type="textarea" v-model="securitySettings.ua.white" :rows="3" placeholder="一行一个UA keyword" />
      </el-form-item>
      
      <div class="divider"></div>
      
      <div class="section-title">Cookie设置</div>
      <el-form-item label="开关">
        <el-switch v-model="securitySettings.cookie.enable" />
      </el-form-item>
      <el-form-item label="作用域" v-if="securitySettings.cookie.enable">
        <el-input v-model="securitySettings.cookie.domain" placeholder="留空则默认为当前域名" />
      </el-form-item>

      <div class="divider"></div>
      
      <div class="section-title">区域屏蔽</div>
      <el-form-item label="区域选择">
        <CountrySelector v-model="securitySettings.regions" />
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import CountrySelector from '@/components/CountrySelector.vue'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  }
})

const emit = defineEmits(['update:modelValue'])

const securitySettings = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>

<style scoped>
.security-config {
  padding: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 20px;
  padding-left: 10px;
  border-left: 3px solid #409eff;
}

.divider {
  height: 1px;
  background-color: #ebeef5;
  margin: 24px 0;
}

.form-helper {
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  margin-top: 6px;
}
</style>