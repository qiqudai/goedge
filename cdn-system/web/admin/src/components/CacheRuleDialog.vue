<template>
  <el-dialog
    :model-value="visible"
    :title="isEditMode ? '编辑缓存规则' : '新增缓存规则'"
    width="600px"
    @update:model-value="$emit('update:visible', $event)"
    :close-on-click-modal="false"
  >
    <el-form label-width="100px">
      <el-form-item label="类型">
        <el-select v-model="form.type" style="width: 100%;">
          <el-option label="首页" value="index" />
          <el-option label="全站" value="all" />
          <el-option label="目录" value="dir" />
          <el-option label="后缀" value="suffix" />
          <el-option label="路径" value="path" />
        </el-select>
      </el-form-item>
      
      <el-form-item label="内容" v-if="['dir', 'suffix', 'path'].includes(form.type)">
        <el-input v-model="form.value" placeholder="支持正则或路径" />
      </el-form-item>
      
      <el-form-item label="有效期">
        <div style="display: flex; gap: 10px; width: 100%;">
           <el-input v-model="form.ttl_value" type="number" style="flex: 1;" placeholder="请输入时长" />
           <el-select v-model="form.ttl_unit" style="width: 80px;">
             <el-option label="秒" value="s" />
             <el-option label="分" value="m" />
             <el-option label="时" value="h" />
             <el-option label="天" value="d" />
           </el-select>
        </div>
      </el-form-item>

      <el-form-item label="忽略参数">
        <el-switch v-model="form.ignore_query" />
      </el-form-item>
      
      <el-form-item label="强制缓存">
        <el-switch v-model="form.force_cache" />
      </el-form-item>

      <!-- Advanced Settings Toggle -->
      <div style="text-align: center; margin-bottom: 10px;">
        <el-button link type="primary" @click="showAdvanced = !showAdvanced">
          {{ showAdvanced ? '收起更多设置' : '展开更多设置' }}
          <el-icon class="el-icon--right">
            <component :is="showAdvanced ? 'ArrowUp' : 'ArrowDown'" />
          </el-icon>
        </el-button>
      </div>

      <!-- Advanced Settings Area -->
      <div v-show="showAdvanced" class="advanced-settings">
        <el-form-item label="分片回源">
           <el-switch v-model="form.enable_slice" />
        </el-form-item>
        
        <el-form-item label="忽略Vary">
           <el-switch v-model="form.ignore_vary" />
        </el-form-item>

        <el-form-item label="不缓存条件">
           <div class="skip-conditions-wrapper">
             <el-table :data="form.skip_conditions" size="small" border style="width: 100%; margin-bottom: 10px;">
               <el-table-column label="匹配项" width="120">
                 <template #default="{ row }">
                   {{ getMatchItemLabel(row.type) }}
                 </template>
               </el-table-column>
               <el-table-column label="匹配值" prop="value" />
               <el-table-column label="操作" width="60" align="center">
                 <template #default="{ $index }">
                   <el-button link type="danger" size="small" @click="removeSkipCondition($index)">删除</el-button>
                 </template>
               </el-table-column>
             </el-table>

             <div class="add-skip-row">
               <el-select v-model="newSkip.type" placeholder="请选择匹配项" size="small" style="width: 130px;">
                 <el-option v-for="opt in skipMatchOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
               </el-select>
               <el-input v-model="newSkip.value" placeholder="请输入匹配值" size="small" style="flex: 1; margin: 0 5px;" />
               <el-button size="small" @click="addSkipCondition">添加</el-button>
             </div>
           </div>
        </el-form-item>
      </div>

    </el-form>
    <template #footer>
      <el-button size="small" @click="$emit('update:visible', false)">取消</el-button>
      <el-button size="small" type="primary" @click="handleSave">保存规则</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { defineProps, defineEmits, ref, watch, computed } from 'vue';
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  rule: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(['update:visible', 'save']);

const defaultFormState = () => ({
  type: 'index',
  value: '',
  ttl: '86400',
  ttl_value: '1',
  ttl_unit: 'd',
  ignore_query: false,
  force_cache: false,
  enable_slice: false,
  ignore_vary: false,
  skip_conditions: []
});

const form = ref(defaultFormState());
const showAdvanced = ref(false);
const isEditMode = computed(() => !!props.rule);

const skipMatchOptions = [
  { label: '请求URI', value: 'request_uri' },
  { label: '请求URI(不带参数)', value: 'uri' },
  { label: '客户IP地址', value: 'ip' },
  { label: '请求协议', value: 'scheme' },
  { label: '请求参数', value: 'args' },
  { label: '域名', value: 'domain' },
  { label: '自定义', value: 'custom' }
]

const newSkip = ref({
  type: 'request_uri',
  value: ''
})

// Helper to convert seconds to unit
const parseTTL = (secondsStr) => {
  let seconds = parseInt(secondsStr || '0')
  if (seconds % 86400 === 0 && seconds !== 0) return { val: seconds / 86400, unit: 'd' }
  if (seconds % 3600 === 0 && seconds !== 0) return { val: seconds / 3600, unit: 'h' }
  if (seconds % 60 === 0 && seconds !== 0) return { val: seconds / 60, unit: 'm' }
  return { val: seconds, unit: 's' }
}

const toSeconds = (val, unit) => {
  const v = parseInt(val || '0')
  if (unit === 'd') return v * 86400
  if (unit === 'h') return v * 3600
  if (unit === 'm') return v * 60
  return v
}

watch(() => props.visible, (newVal) => {
  if (newVal) {
    if (props.rule) {
      const parsed = parseTTL(props.rule.ttl)
      form.value = { 
        ...defaultFormState(),
        ...props.rule,
        ttl_value: parsed.val,
        ttl_unit: parsed.unit,
        skip_conditions: JSON.parse(JSON.stringify(props.rule.skip_conditions || []))
      };
    } else {
      form.value = defaultFormState();
    }
  }
});

const getMatchItemLabel = (val) => {
  return skipMatchOptions.find(o => o.value === val)?.label || val
}

const addSkipCondition = () => {
  if (!newSkip.value.value) return 
  form.value.skip_conditions.push({ ...newSkip.value })
  newSkip.value.value = ''
}

const removeSkipCondition = (idx) => {
  form.value.skip_conditions.splice(idx, 1)
}

const handleSave = () => {
  // Calculate final TTL
  const finalTTL = toSeconds(form.value.ttl_value, form.value.ttl_unit)
  
  emit('save', { 
    ...form.value,
    ttl: String(finalTTL)
  });
  emit('update:visible', false);
};
</script>

<style scoped>
.advanced-settings {
  background-color: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
}
.skip-conditions-wrapper {
  background: #fff;
  padding: 10px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
}
.add-skip-row {
  display: flex;
  align-items: center;
  margin-top: 5px;
}
</style>
