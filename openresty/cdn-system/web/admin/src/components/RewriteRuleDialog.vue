<template>
  <el-dialog
    :model-value="visible"
    title="新增/编辑重写规则"
    width="500px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <el-form label-width="100px">
      <el-form-item label="匹配URI">
        <el-input v-model="form.match" placeholder="(.*)" />
      </el-form-item>
      <el-form-item label="重写到">
        <el-input v-model="form.replace" placeholder="https://www.baidu.com$1" />
      </el-form-item>
      <el-form-item label="响应码">
        <el-select v-model="form.code">
          <el-option value="301" label="301 (永久移动)" />
          <el-option value="302" label="302 (临时移动)" />
          <el-option value="307" label="307 (临时重定向)" />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:visible', false)">取消</el-button>
      <el-button type="primary" @click="handleSave">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch, defineProps, defineEmits } from 'vue';
import { ElMessage } from 'element-plus';

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
  match: '',
  replace: '',
  code: '301',
});

const form = ref(defaultFormState());

watch(() => props.visible, (newVal) => {
  if (newVal) {
    form.value = props.rule ? { ...props.rule } : defaultFormState();
  }
});

const handleSave = () => {
  if (!form.value.match || !form.value.replace) {
    ElMessage.warning('请填写完整信息');
    return;
  }
  emit('save', { ...form.value });
  emit('update:visible', false);
};
</script>
