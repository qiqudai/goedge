<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="600px"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="匹配URI" prop="match">
        <el-input v-model="form.match" placeholder="(.*)" />
      </el-form-item>
      
      <el-form-item label="重写到" prop="redirect">
        <el-input v-model="form.redirect" placeholder="https://www.baidu.com$1" />
      </el-form-item>
      
      <el-form-item label="响应码" prop="code">
        <el-select v-model="form.code" placeholder="请选择响应码">
          <el-option label="301" value="301" />
          <el-option label="302" value="302" />
          <el-option label="307" value="307" />
          <el-option label="内部" value="internal" />
        </el-select>
      </el-form-item>
      
      <el-form-item label="条件设置">
        <div class="condition-section">
          <div class="condition-toggle" @click="conditionsExpanded = !conditionsExpanded">
            <span>{{ conditionsExpanded ? '▼ 收起设置转向条件' : '▶ 展开设置转向条件' }}</span>
          </div>
          
          <div v-show="conditionsExpanded" class="condition-content">
            <div class="condition-selector">
              <el-select 
                v-model="selectedCondition" 
                placeholder="选择条件类型" 
                style="width: 200px;"
                @change="addCondition"
              >
                <el-option
                  v-for="opt in conditionOptions"
                  :key="opt.value"
                  :label="opt.label"
                  :value="opt.value"
                  :disabled="selectedConditions.includes(opt.value)"
                />
              </el-select>
            </div>
            
            <div class="selected-conditions">
              <div 
                v-for="condition in selectedConditionsData" 
                :key="condition.key" 
                class="condition-item"
              >
                <label>{{ condition.label }}:</label>
                <el-input 
                  v-model="condition.value" 
                  :placeholder="condition.placeholder"
                  style="width: 200px; margin-left: 10px;"
                />
                <el-button 
                  link 
                  type="danger" 
                  size="small" 
                  @click="removeCondition(condition.key)"
                  style="margin-left: 10px;"
                >
                  CloseBtn
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </el-form-item>
    </el-form>
    
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, reactive } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  rule: { type: Object, default: null }
})

const emit = defineEmits(['update:modelValue', 'submit'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const title = computed(() => props.rule ? '编辑转向' : '新增转向')

const formRef = ref()

const form = reactive({
  match: '(.*)',
  redirect: 'https://www.baidu.com$1',
  code: '301',
  conditions: []
})

const rules = {
  match: [{ required: true, message: '请输入匹配URI', trigger: 'blur' }],
  redirect: [{ required: true, message: '请输入重写地址', trigger: 'blur' }],
  code: [{ required: true, message: '请选择响应码', trigger: 'change' }]
}

const conditionOptions = [
  { value: 'accept_language', label: 'Accept-Language', placeholder: 'zh-CN|en-US' },
  { value: 'province', label: '省份', placeholder: '广西|广东' },
  { value: 'domain_port', label: '域名:端口', placeholder: 'www.aaa.com|www.bbb.com' },
  { value: 'user_agent', label: 'User-Agent', placeholder: 'Safari|Chrome' },
  { value: 'referer', label: 'Referer', placeholder: 'www.qq.com|www.baidu.com' },
  { value: 'country_code', label: '国家代码', placeholder: 'cn|us' },
  { value: 'city', label: '城市', placeholder: '宁波|十堰' },
  { value: 'isp', label: '运营商', placeholder: '电信|阿里云|腾讯' },
  { value: 'asn', label: 'AS号码', placeholder: '45102|45103' }
]

const conditionsExpanded = ref(false)
const selectedCondition = ref('')

// 已选中的条件
const selectedConditions = ref([])

// 已选中条件的详细信息
const selectedConditionsData = computed(() => {
  return selectedConditions.value.map(key => {
    const option = conditionOptions.find(opt => opt.value === key)
    return {
      key,
      label: option?.label || key,
      placeholder: option?.placeholder || '',
      value: form.conditions.find(c => c.key === key)?.value || ''
    }
  })
})

watch(() => props.rule, (newRule) => {
  if (newRule) {
    form.match = newRule.match || '(.*)'
    form.redirect = newRule.redirect || ''
    form.code = newRule.code || '301'
    form.conditions = newRule.conditions || []
    selectedConditions.value = form.conditions.map(c => c.key)
  } else {
    form.match = '(.*)'
    form.redirect = 'https://www.baidu.com$1'
    form.code = '301'
    form.conditions = []
    selectedConditions.value = []
  }
}, { immediate: true })

const addCondition = () => {
  if (selectedCondition.value && !selectedConditions.value.includes(selectedCondition.value)) {
    selectedConditions.value.push(selectedCondition.value)
    form.conditions.push({
      key: selectedCondition.value,
      value: ''
    })
    selectedCondition.value = ''
  }
}

const removeCondition = (key) => {
  selectedConditions.value = selectedConditions.value.filter(k => k !== key)
  form.conditions = form.conditions.filter(c => c.key !== key)
}

const handleSubmit = () => {
  formRef.value?.validate((valid) => {
    if (valid) {
      const ruleData = {
        ...form,
        conditions: form.conditions.filter(c => c.value.trim())
      }
      emit('submit', ruleData)
      handleClose()
    }
  })
}

const handleClose = () => {
  visible.value = false
  selectedCondition.value = ''
  conditionsExpanded.value = false
}
</script>

<style scoped>
.condition-section {
  width: 100%;
}

.condition-toggle {
  cursor: pointer;
  color: #606266;
  font-size: 14px;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  background: #f5f7fa;
  padding: 8px 12px;
  border-radius: 4px;
  transition: all 0.3s;
}

.condition-toggle:hover {
  background: #edf2f7;
  color: #409eff;
}

.condition-content {
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 16px;
  background-color: #fafafa;
}

.condition-selector {
  margin-bottom: 16px;
}

.selected-conditions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.condition-item {
  display: flex;
  align-items: center;
  background: white;
  padding: 12px;
  border-radius: 4px;
  border: 1px solid #e4e7ed;
}

.condition-item label {
  min-width: 80px;
  font-weight: 500;
  color: #303133;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>