<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :width="width"
    :close-on-click-modal="false"
    :close-on-press-escape="!running"
    :show-close="!running"
    align-center
    @closed="handleClosed"
  >
    <div class="batch-progress-body">
      <div class="batch-progress-row">
        <span class="batch-progress-label">{{ statusLabel }}</span>
        <span class="batch-progress-percent">{{ percentage }}%</span>
      </div>
      <el-progress
        :percentage="percentage"
        :status="progressStatus"
        :stroke-width="14"
        :show-text="false"
      />
      <div class="batch-progress-stats">
        <span class="stat-item stat-pending">等待: {{ stats.pending }}</span>
        <span class="stat-item stat-running">运行: {{ stats.running }}</span>
        <span class="stat-item stat-success">成功: {{ stats.success }}</span>
        <span class="stat-item stat-fail">失败: {{ stats.fail }}</span>
        <span class="stat-item stat-total">总计: {{ stats.total }}</span>
      </div>
      <div v-if="failItems.length" class="batch-progress-fails">
        <div class="fails-title">失败列表</div>
        <el-table :data="failItems" size="small" border max-height="240" style="width: 100%">
          <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
          <el-table-column prop="reason" label="原因" min-width="220" show-overflow-tooltip />
        </el-table>
      </div>
    </div>
    <template #footer>
      <el-button v-if="running" type="warning" @click="handleCancel">取消</el-button>
      <el-button v-else type="primary" @click="handleClose">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'

const props = defineProps({
  modelValue: Boolean,
  title: {
    type: String,
    default: '操作进度'
  },
  width: {
    type: String,
    default: '520px'
  },
  total: {
    type: Number,
    default: 0
  },
  done: {
    type: Number,
    default: 0
  },
  success: {
    type: Number,
    default: 0
  },
  fail: {
    type: Number,
    default: 0
  },
  running: {
    type: Boolean,
    default: false
  },
  failItems: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue', 'cancel', 'closed'])

const visible = ref(false)
let cancelRequested = false

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (!val) cancelRequested = false
}, { immediate: true })

watch(visible, (val) => emit('update:modelValue', val))

const stats = computed(() => ({
  total: props.total || 0,
  pending: Math.max(0, (props.total || 0) - (props.done || 0) - (props.running ? 0 : 0)),
  running: props.running ? Math.max(0, (props.total || 0) - (props.done || 0)) : 0,
  success: props.success || 0,
  fail: props.fail || 0
}))

const percentage = computed(() => {
  if (props.total <= 0) return 0
  return Math.min(100, Math.round((props.done / props.total) * 100))
})

const progressStatus = computed(() => {
  if (percentage.value < 100) return ''
  return props.fail > 0 ? 'exception' : 'success'
})

const statusLabel = computed(() => {
  if (cancelRequested) return '正在取消...'
  if (percentage.value >= 100) return props.fail > 0 ? '操作完成（部分失败）' : '操作完成'
  if (props.total > 0) return `正在处理: ${props.done} / ${props.total}`
  return '准备中...'
})

const handleCancel = () => {
  cancelRequested = true
  emit('cancel')
}

const handleClose = () => {
  visible.value = false
}

const handleClosed = () => {
  emit('closed')
}

onBeforeUnmount(() => {
  cancelRequested = false
})
</script>

<style scoped>
.batch-progress-body {
  padding: 4px 0 8px;
}
.batch-progress-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--el-text-color-primary, #303133);
}
.batch-progress-percent {
  font-weight: 600;
  color: var(--el-color-primary, #409eff);
}
.batch-progress-stats {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 12px;
}
.stat-item { color: var(--el-text-color-regular, #606266); }
.stat-pending { color: var(--el-text-color-secondary, #909399); }
.stat-running { color: var(--el-color-primary, #409eff); }
.stat-success { color: var(--el-color-success, #67c23a); }
.stat-fail { color: var(--el-color-danger, #f56c6c); }
.stat-total { color: var(--el-text-color-secondary, #909399); }

.batch-progress-fails {
  margin-top: 14px;
}
.fails-title {
  font-size: 12px;
  color: var(--el-color-danger, #f56c6c);
  margin-bottom: 6px;
  font-weight: 600;
}
</style>
