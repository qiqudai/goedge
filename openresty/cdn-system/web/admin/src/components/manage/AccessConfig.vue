<template>
  <div class="access-config">
    <el-form label-width="150px" class="config-form">
      <div class="section-title">ACL设置</div>
      <el-form-item label="ACL选择" style="width: 520px">
        <el-select v-model="accessSettings.acl" placeholder="请选择" style="width: 100%" clearable @change="handleSave">
          <el-option
            v-for="item in aclList"
            :key="item.id"
            :label="item.name"
            :value="item.id"
          />
        </el-select>
        <div class="form-helper">需要到左侧菜单规则管理里创建好ACL，再在这里选择应用</div>
      </el-form-item>

      <div class="divider"></div>
      
      <div class="section-title">防盗链设置</div>
      <el-form-item label="开关">
        <el-switch v-model="accessSettings.hotlink.enable" @change="handleSave" />
      </el-form-item>
      <template v-if="accessSettings.hotlink.enable">
        <el-form-item label="防盗链范围">
          <div style="display: flex; align-items: center; gap: 10px; flex-wrap: wrap;">
            <el-radio-group v-model="accessSettings.hotlink.scope" @change="handleSave">
              <el-radio value="all">整站</el-radio>
              <el-radio value="suffix">后缀</el-radio>
              <el-radio value="dir">目录</el-radio>
              <el-radio value="path">单个路径</el-radio>
            </el-radio-group>
            <el-input
              v-if="accessSettings.hotlink.scope !== 'all'"
              v-model="accessSettings.hotlink.value"
              style="width: 300px;"
              :placeholder="getHotlinkPlaceholder()"
              @change="saveSettings(true)"
            />
          </div>
        </el-form-item>
        <el-form-item label="允许空来源">
          <el-radio-group v-model="accessSettings.hotlink.allowEmpty" @change="handleSave">
            <el-radio :value="true">允许</el-radio>
            <el-radio :value="false">不允许</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="额外允许域名" style="width: 520px">
          <el-input v-model="accessSettings.hotlink.domains" placeholder="请输入除当前网站域名之外的域名 多个域名空格分隔" @change="saveSettings(true)" />
        </el-form-item>
      </template>

      <div class="divider"></div>

      <div class="section-title">跨域访问设置</div>
      <el-form-item label="开关">
        <el-switch v-model="accessSettings.cors.enable" @change="handleSave" />
      </el-form-item>
      <template v-if="accessSettings.cors.enable">
        <div class="cors-more-toggle" @click="corsExpanded = !corsExpanded">
          <span>{{ corsExpanded ? '▼ 收起更多设置' : '▶ 查看更多设置' }}</span>
        </div>
        
        <div v-show="corsExpanded">
          <el-form-item label="allow_origin" style="width: 520px">
            <el-input v-model="accessSettings.cors.allowOrigin" @change="saveSettings(true)" />
          </el-form-item>
          <el-form-item label="allow_methods" style="width: 520px">
            <el-input v-model="accessSettings.cors.allowMethods" @change="saveSettings(true)" />
          </el-form-item>
          <el-form-item label="allow_headers" style="width: 520px">
            <el-input v-model="accessSettings.cors.allowHeaders" @change="saveSettings(true)" />
          </el-form-item>
          <el-form-item label="expose_headers" style="width: 520px">
            <el-input v-model="accessSettings.cors.exposeHeaders" @change="saveSettings(true)" />
          </el-form-item>
          <el-form-item label="allow_credentials" style="width: 520px">
            <el-radio-group v-model="accessSettings.cors.allowCredentials" @change="handleSave">
              <el-radio :value="true">允许</el-radio>
              <el-radio :value="false">不允许</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="max_age" style="width: 520px">
            <el-input v-model="accessSettings.cors.maxAge" @change="saveSettings(true)" />
          </el-form-item>
        </div>
      </template>
    </el-form>
  </div>
</template>



<script setup>
import { ref, watch, computed } from 'vue'
import { getHotlinkPlaceholder } from '@/utils/siteHelpers'
import { useSiteSettings } from '@/composables/useSiteSettings'

const props = defineProps({
  modelValue: { 
    type: Object, 
    required: true 
  },
  aclList: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

const { saveSettings } = useSiteSettings()

const localSettings = ref({
  acl: props.modelValue?.acl || '',
  hotlink: {
    enable: props.modelValue?.hotlink?.enable || false,
    scope: props.modelValue?.hotlink?.scope || 'all',
    value: props.modelValue?.hotlink?.value || '',
    allowEmpty: props.modelValue?.hotlink?.allowEmpty !== false,
    domains: props.modelValue?.hotlink?.domains || ''
  },
  cors: {
    enable: props.modelValue?.cors?.enable || false,
    allowOrigin: props.modelValue?.cors?.allowOrigin || '*',
    allowMethods: props.modelValue?.cors?.allowMethods || 'GET,POST,OPTIONS',
    allowHeaders: props.modelValue?.cors?.allowHeaders || '*',
    exposeHeaders: props.modelValue?.cors?.exposeHeaders || '',
    allowCredentials: props.modelValue?.cors?.allowCredentials || false,
    maxAge: props.modelValue?.cors?.maxAge || '3600'
  }
})

let isInternalUpdate = false

const handleSave = () => {
    saveSettings(true)
}

watch(localSettings, (newVal) => {
  isInternalUpdate = true
  emit('update:modelValue', {
    ...props.modelValue,
    ...newVal
  })
}, { deep: true })

watch(() => props.modelValue, (newVal) => {
  if (newVal && !isInternalUpdate) {
    localSettings.value = {
      acl: newVal.acl || '',
      hotlink: {
        enable: newVal.hotlink?.enable || false,
        scope: newVal.hotlink?.scope || 'all',
        value: newVal.hotlink?.value || '',
        allowEmpty: newVal.hotlink?.allowEmpty !== false,
        domains: newVal.hotlink?.domains || ''
      },
      cors: {
        enable: newVal.cors?.enable || false,
        allowOrigin: newVal.cors?.allowOrigin || '*',
        allowMethods: newVal.cors?.allowMethods || 'GET,POST,OPTIONS',
        allowHeaders: newVal.cors?.allowHeaders || '*',
        exposeHeaders: newVal.cors?.exposeHeaders || '',
        allowCredentials: newVal.cors?.allowCredentials || false,
        maxAge: newVal.cors?.maxAge || '3600'
      }
    }
  }
  isInternalUpdate = false
}, { deep: true })

const accessSettings = localSettings
const corsExpanded = ref(false)
const aclList = computed(() => props.aclList)
</script>

<style scoped>
.access-config {
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

.cors-more-toggle {
  cursor: pointer;
  color: #606266;
  font-size: 14px;
  margin-bottom: 20px;
  margin-left: 150px;
  display: flex;
  align-items: center;
  background: #f5f7fa;
  padding: 10px 15px;
  border-radius: 4px;
  transition: all 0.3s;
}

.cors-more-toggle:hover {
  background: #edf2f7;
  color: #409eff;
}
</style>