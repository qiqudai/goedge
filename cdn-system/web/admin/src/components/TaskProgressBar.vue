<template>
  <div class="task-progress-bar" v-if="batchId">
    <div class="flex flex-col gap-2">
       <div class="flex justify-between text-sm">
         <span>正在处理: {{ progress.running }} / {{ progress.total }}</span>
         <span>进度: {{ percentage }}%</span>
       </div>
       <el-progress :percentage="percentage" :status="progressStatus"></el-progress>
       
       <div class="stats mt-2 flex gap-4 text-sm">
         <span class="text-gray-500">等待: {{ progress.pending }}</span>
         <span class="text-blue-500">运行: {{ progress.running }}</span>
         <span class="text-green-500">成功: {{ progress.success }}</span>
         <span class="text-red-500">失败: {{ progress.fail }}</span>
       </div>

       <div v-if="progress.fail_items && progress.fail_items.length > 0" class="mt-4">
         <p class="text-red-500 font-bold mb-2">失败列表:</p>
         <el-table :data="progress.fail_items" size="small" border style="width: 100%" max-height="250">
           <el-table-column prop="domain" label="域名" width="200"></el-table-column>
           <el-table-column prop="reason" label="原因" show-overflow-tooltip></el-table-column>
         </el-table>
       </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted, computed } from 'vue'
import request from '@/utils/request'

const props = defineProps({
  batchId: {
    type: String,
    default: ''
  },
  apiUrl: {
    type: String, // e.g., /certs/batch/{id}/progress -> template /certs/batch/${id}/progress
    required: true
  }
})

const emit = defineEmits(['completed'])

const progress = ref({
  total: 0,
  done: 0,
  success: 0,
  fail: 0,
  running: 0,
  pending: 0,
  fail_items: []
})
const percentage = ref(0)
const timer = ref(null)

const progressStatus = computed(() => {
  if (percentage.value === 100) {
    return progress.value.fail > 0 ? 'exception' : 'success'
  }
  return ''
})

const fetchProgress = async () => {
    if (!props.batchId) return
    try {
        const url = props.apiUrl.replace(':id', props.batchId)
        const res = await request.get(url)
        // Ensure res.data or res contains the payload
        const data = res.data || res
        
        progress.value = {
            total: data.total || 0,
            done: data.done || 0,
            success: data.success || 0,
            fail: data.fail || 0,
            running: data.running || 0,
            pending: data.pending || 0,
            fail_items: data.fail_items || []
        }
        
        let pct = 0
        if (data.total > 0) {
            pct = Math.round((data.done / data.total) * 100)
        }
        percentage.value = pct

        if (data.done >= data.total && data.total > 0) {
             stopPolling()
             emit('completed')
        }
    } catch (e) {
        console.error("Fetch progress failed", e)
    }
}

const startPolling = () => {
  stopPolling()
  fetchProgress()
  timer.value = setInterval(fetchProgress, 2000)
}

const stopPolling = () => {
  if (timer.value) {
    clearInterval(timer.value)
    timer.value = null
  }
}

watch(() => props.batchId, (val) => {
  if (val) {
    startPolling()
  } else {
    stopPolling()
  }
}, { immediate: true })

onUnmounted(() => {
  stopPolling()
})
</script>
