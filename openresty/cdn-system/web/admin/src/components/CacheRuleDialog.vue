<template>
  <el-dialog
    :model-value="visible"
    :title="isEditMode ? '编辑缓存规则' : '新增缓存规则'"
    width="520px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <el-form label-width="120px">
      <el-form-item label="类型">
        <el-select v-model="form.type">
          <el-option label="首页" value="index" />
          <el-option label="全站" value="all" />
          <el-option label="目录" value="dir" />
          <el-option label="后缀" value="suffix" />
          <el-option label="路径" value="path" />
        </el-select>
      </el-form-item>
      <el-form-item label="内容">
        <el-input v-model="form.value" placeholder="支持正则或路径" />
      </el-form-item>
      <el-form-item label="TTL">
        <el-input v-model="form.ttl" placeholder="单位：秒" />
      </el-form-item>
      <el-form-item label="忽略参数">
        <el-switch v-model="form.ignore_query" />
      </el-form-item>
      <el-form-item label="强制缓存">
        <el-switch v-model="form.force_cache" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button size="small" @click="$emit('update:visible', false)">取消</el-button>
      <el-button size="small" type="primary" @click="handleSave">保存规则</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { defineProps, defineEmits, ref, watch, computed } from 'vue';

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
  ignore_query: false,
  force_cache: false,
});

const form = ref(defaultFormState());
const isEditMode = computed(() => !!props.rule);

watch(() => props.visible, (newVal) => {
  if (newVal) {
    if (props.rule) {
      form.value = { ...props.rule };
    } else {
      form.value = defaultFormState();
    }
  }
});

const handleSave = () => {
  emit('save', { ...form.value });
  emit('update:visible', false);
};
</script>
