<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="780px"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <div v-if="loading" class="text-gray-500">正在加载任务...</div>

    <template v-else-if="task">
      <el-descriptions :column="2" border class="mb-3">
        <el-descriptions-item label="任务ID">{{ task.id }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ task.type || '-' }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ task.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="stateTagType(task.state)">{{ stateText(task.state) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ formatTime(task.create_at) }}</el-descriptions-item>
        <el-descriptions-item label="开始时间">{{ formatTime(task.start_at) }}</el-descriptions-item>
        <el-descriptions-item label="结束时间">{{ formatTime(task.end_at) }}</el-descriptions-item>
        <el-descriptions-item label="失败次数">{{ task.err_times ?? 0 }}</el-descriptions-item>
      </el-descriptions>

      <div class="mb-3">
        <div class="flex justify-between text-sm mb-1">
          <span>节点进度: {{ counts.done }} / {{ counts.total }}</span>
          <span>成功: {{ counts.success }} 失败: {{ counts.fail }} 运行中: {{ counts.running }}</span>
        </div>
        <el-progress :percentage="percentage" :status="percentageStatus" />
      </div>

      <el-table :data="nodeRows" size="small" border style="width: 100%" max-height="260">
        <el-table-column prop="node_id" label="节点" width="140" />
        <el-table-column prop="state" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="stateTagType(row.state)">{{ stateText(row.state) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="最近结果" show-overflow-tooltip />
        <el-table-column prop="time" label="时间" width="170" />
      </el-table>

      <el-collapse class="mt-3" v-if="logRows.length">
        <el-collapse-item title="任务日志" name="log">
          <el-table :data="logRows" size="small" border style="width: 100%" max-height="260">
            <el-table-column prop="time" label="时间" width="170" />
            <el-table-column prop="node_id" label="节点" width="140" />
            <el-table-column prop="state" label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="stateTagType(row.state)">{{ stateText(row.state) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="内容" show-overflow-tooltip />
          </el-table>
        </el-collapse-item>
      </el-collapse>
    </template>

    <div v-else class="text-gray-500">暂无任务数据</div>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import request from '@/utils/request'

const props = defineProps({
  modelValue: Boolean,
  taskId: {
    type: [Number, String],
    default: ''
  },
  title: {
    type: String,
    default: '任务详情'
  }
})

const emit = defineEmits(['update:modelValue', 'completed'])

const visible = ref(false)
const loading = ref(false)
const task = ref(null)
const timer = ref(null)

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    startPolling()
  } else {
    stopPolling()
  }
})

watch(() => visible.value, (val) => {
  emit('update:modelValue', val)
})

watch(() => props.taskId, () => {
  if (visible.value) {
    startPolling()
  }
})

const formatTime = (val) => {
  if (!val) return '-'
  const s = String(val)
  if (s.startsWith('0001')) return '-'
  return s.replace('T', ' ').substring(0, 19)
}

const safeParseJSON = (raw) => {
  if (!raw || typeof raw !== 'string') return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

const progressMap = computed(() => {
  const obj = safeParseJSON(task.value?.progress) || {}
  return obj && typeof obj === 'object' ? obj : {}
})

const logRows = computed(() => {
  const raw = task.value?.ret
  const parsed = safeParseJSON(raw)
  if (Array.isArray(parsed)) {
    return parsed.map((x) => ({
      time: x.time || '-',
      node_id: x.node_id || '-',
      state: x.state || '-',
      message: x.message || '',
      attempt: x.attempt ?? 0
    }))
  }
  if (typeof raw === 'string' && raw.trim()) {
    return [{ time: '-', node_id: '-', state: 'log', message: raw, attempt: 0 }]
  }
  return []
})

const nodeRows = computed(() => {
  const lastByNode = {}
  for (const row of logRows.value) {
    if (!row.node_id || row.node_id === '-') continue
    lastByNode[row.node_id] = row
  }
  return Object.keys(progressMap.value).sort().map((nodeId) => {
    const state = progressMap.value[nodeId]
    const last = lastByNode[nodeId]
    return {
      node_id: nodeId,
      state,
      message: last?.message || '',
      time: last?.time || '-'
    }
  })
})

const counts = computed(() => {
  const values = Object.values(progressMap.value || {})
  const total = values.length
  const success = values.filter(v => v === 'done' || v === 'success').length
  const fail = values.filter(v => v === 'fail').length
  const running = values.filter(v => v === 'running').length
  const done = success + fail
  return { total, success, fail, running, done }
})

const percentage = computed(() => {
  if (counts.value.total <= 0) return 0
  return Math.round((counts.value.done / counts.value.total) * 100)
})

const percentageStatus = computed(() => {
  if (percentage.value !== 100) return ''
  return counts.value.fail > 0 ? 'exception' : 'success'
})

const stateText = (val) => {
  const map = {
    waiting: '等待中',
    running: '执行中',
    done: '完成',
    success: '成功',
    fail: '失败'
  }
  return map[val] || val || '-'
}

const stateTagType = (val) => {
  const map = {
    waiting: 'info',
    running: 'warning',
    done: 'success',
    success: 'success',
    fail: 'danger'
  }
  return map[val] || 'info'
}

const fetchTask = async () => {
  const id = String(props.taskId || '').trim()
  if (!id) return
  loading.value = true
  try {
    const res = await request.get(`/tasks/${encodeURIComponent(id)}`, { skipLoading: true })
    task.value = res.data || res
  } finally {
    loading.value = false
  }

  const state = String(task.value?.state || '').toLowerCase()
  if (state === 'done' || state === 'fail') {
    stopPolling()
    emit('completed', task.value)
  }
}

const startPolling = () => {
  stopPolling()
  fetchTask()
  timer.value = setInterval(fetchTask, 2000)
}

const stopPolling = () => {
  if (timer.value) {
    clearInterval(timer.value)
    timer.value = null
  }
}

const handleClosed = () => {
  stopPolling()
  task.value = null
}

onUnmounted(() => stopPolling())
</script>

<style scoped>
.mb-1 { margin-bottom: 4px; }
.mb-3 { margin-bottom: 12px; }
.mt-3 { margin-top: 12px; }
.text-sm { font-size: 12px; }
.text-gray-500 { color: #909399; }
.flex { display: flex; }
.justify-between { justify-content: space-between; }
</style>

