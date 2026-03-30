<template>
  <el-dialog v-model="visible" title="监控日志" width="680px">
    <el-form :inline="true" class="monitor-form">
      <el-form-item label="日志查看">
        <el-select v-model="query.type" style="width: 200px;">
          <el-option label="可用性监控日志" value="availability" />
          <el-option label="可用性切换日志" value="availability_switch" />
          <el-option label="带宽监控日志" value="bandwidth" />
          <el-option label="带宽切换日志" value="bandwidth_switch" />
        </el-select>
      </el-form-item>
      <el-form-item label="时间段">
        <el-date-picker
          v-model="query.timeRange"
          type="datetimerange"
          range-separator="至"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          value-format="YYYY-MM-DD HH:mm:ss"
          clearable
          style="width: 260px;"
        />
      </el-form-item>
      <el-form-item label="监控组">
        <el-select v-model="query.group" style="width: 200px;">
          <el-option label="所有监控组" value="all" />
        </el-select>
      </el-form-item>
    </el-form>
    <AppTable
      v-loading="loading"
      :data="list"
      v-model:current-page="query.page"
      v-model:page-size="query.pageSize"
      persist-key="node-monitor-logs"
      layout="total, sizes, prev, pager, next"
      :total="total"
      @current-change="fetchData"
      @size-change="fetchData"
      border
    >
      <el-table-column prop="checked_at" label="检测时间" min-width="140" />
      <el-table-column prop="fail_count" label="失败个数" width="100" align="center" />
      <el-table-column prop="total_count" label="总检测点" width="100" align="center" />
    </AppTable>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import request from '@/utils/request'

const props = defineProps({
  nodeId: Number,
  modelValue: Boolean
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(false)
const list = ref([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ type: 'availability', group: 'all', timeRange: [], page: 1, pageSize: 10 })

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val && props.nodeId) fetchData()
})
watch(visible, (val) => emit('update:modelValue', val))
watch(() => query.type, () => {
  query.page = 1
  if (visible.value) fetchData()
})
watch(() => query.group, () => {
  query.page = 1
  if (visible.value) fetchData()
})
watch(() => query.timeRange, () => {
  query.page = 1
  if (visible.value) fetchData()
}, { deep: true })

const fetchData = async () => {
  loading.value = true
  try {
    const { data } = await request.get(`/nodes/${props.nodeId}/monitor_logs`, { params: query })
    list.value = data.list || []
    total.value = data.total || 0
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.monitor-form :deep(.el-form-item__label) {
  color: var(--el-text-color-regular);
}
</style>
