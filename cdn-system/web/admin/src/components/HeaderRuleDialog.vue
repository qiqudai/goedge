<template>
  <el-dialog
    :model-value="visible"
    :title="title"
    width="500px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <el-form label-width="100px">
      <el-form-item label="名称">
        <el-input v-model="form.name" placeholder="如：X-Real-IP" />
      </el-form-item>
      <el-form-item label="值">
        <el-input v-model="form.value" placeholder="请输入值" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:visible', false)">取消</el-button>
      <el-button type="primary" @click="handleSave">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch, computed, defineProps, defineEmits } from 'vue';
import { ElMessage } from 'element-plus';

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  type: {
    type: String,
    default: 'req', // 'req' or 'res'
  },
  rule: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(['update:visible', 'save']);

const title = computed(() => {
  const action = props.rule ? '编辑' : '新增';
  const typeName = props.type === 'req' ? '请求头' : '响应头';
  return `${action}${typeName}`;
});

const defaultFormState = () => ({
  name: '',
  value: '',
});

const form = ref(defaultFormState());

watch(() => props.visible, (newVal) => {
  if (newVal) {
    form.value = props.rule ? { ...props.rule } : defaultFormState();
  }
});

const handleSave = () => {
  if (!form.value.name) {
    ElMessage.warning('请填写名称');
    return;
  }
  emit('save', { ...form.value });
  emit('update:visible', false);
};
</script>
