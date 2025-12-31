<template>
  <div class="notify-item-config">
    <div class="section-divider">
      <span>{{ title }}</span>
    </div>

    <div class="item-body">
      <el-form-item label="开关">
        <el-switch v-model="localConfig.enable" :active-value="1" :inactive-value="0" />
      </el-form-item>

      <div v-if="localConfig.enable === 1">
        <el-form-item label="通知方式">
          <el-checkbox-group v-model="localConfig.methods">
            <el-checkbox label="email">电子邮件</el-checkbox>
            <el-checkbox label="sms">手机短信</el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <!-- Custom Extra Fields (Thresholds etc) -->
        <slot name="extra"></slot>

        <el-form-item label="连续通知次数">
          <el-input-number v-model="localConfig.continuous_times" :min="1" controls-position="right" style="width: 100%" />
        </el-form-item>

        <el-form-item label="间隔时间">
           <el-input v-model.number="localConfig.interval" style="width: 100%;">
             <template #append>小时</template>
           </el-input>
        </el-form-item>

        <el-form-item label="通知模板">
          <el-radio-group v-model="templateType">
            <el-radio label="email">邮件模板</el-radio>
            <el-radio label="sms">短信模板</el-radio>
          </el-radio-group>
        </el-form-item>

        <div v-if="templateType === 'email'" class="template-box">
          <el-form-item label="标题" label-width="60px">
            <el-input v-model="localConfig.email_template.title" placeholder="请输入邮件标题" />
          </el-form-item>
          <el-form-item label="内容" label-width="60px">
            <el-input type="textarea" v-model="localConfig.email_template.content" :rows="5" placeholder="请输入HTML内容" />
             <div class="variables-tip" v-if="variables && variables.length">
               可用变量: <span v-for="v in variables" :key="v" class="code-tag" @click="insertVar('email', v)">{{ v }}</span>
             </div>
          </el-form-item>
        </div>

        <div v-if="templateType === 'sms'" class="template-box">
          <el-form-item label="内容" label-width="60px">
            <el-input type="textarea" v-model="localConfig.sms_template.content" :rows="5" placeholder="请输入短信内容" />
             <div class="variables-tip" v-if="variables && variables.length">
               可用变量: <span v-for="v in variables" :key="v" class="code-tag" @click="insertVar('sms', v)">{{ v }}</span>
             </div>
          </el-form-item>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  },
  title: String,
  variables: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

const localConfig = ref({
  enable: 0,
  methods: [],
  continuous_times: 1,
  interval: 24, // Default 24 hours
  email_template: { title: '', content: '' },
  sms_template: { content: '' }
})

const templateType = ref('email')

watch(() => props.modelValue, (val) => {
  if (val) {
    // Merge defaults
    localConfig.value = {
      enable: 0,
      methods: [],
      continuous_times: 1,
      interval: 24,
      email_template: { title: '', content: '' },
      sms_template: { content: '' },
      ...JSON.parse(JSON.stringify(val))
    }
    // Ensure nested objects
    if (!localConfig.value.methods) localConfig.value.methods = []
    if (!localConfig.value.email_template) localConfig.value.email_template = { title: '', content: '' }
    if (!localConfig.value.sms_template) localConfig.value.sms_template = { content: '' }
  }
}, { immediate: true, deep: true })

watch(localConfig, (val) => {
  emit('update:modelValue', val)
}, { deep: true })

const insertVar = (type, v) => {
  // Simple append for now, could be cursor insertion
  if (type === 'email') {
    localConfig.value.email_template.content += v
  } else {
    localConfig.value.sms_template.content += v
  }
}

</script>

<style scoped>
.notify-item-config {
  margin-bottom: 30px;
}
.section-divider {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  border-bottom: 1px solid #ebeef5;
  padding-bottom: 10px;
}
.section-divider span {
    padding-left: 10px;
}
.item-body {
    padding-left: 20px; 
    max-width: 800px;
}

.template-box {
    background: #fdfdfd; 
    padding: 15px; 
    border: 1px solid #f0f0f0; 
    border-radius: 4px;
    margin-top: 5px;
}

.code-tag {
  background: #ecf5ff;
  border: 1px solid #d9ecff;
  color: #409eff;
  padding: 2px 6px;
  border-radius: 4px;
  margin-right: 5px;
  font-size: 12px;
  cursor: pointer;
}
.variables-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}
</style>
