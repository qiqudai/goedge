<template>
  <div class="domain-batch-input">
    <el-input
      v-model="rawInput"
      type="textarea"
      :rows="10"
      placeholder="请输入域名，每行一个，或者用逗号/空格分隔。
例如：
example.com
sub.example.com
test.com"
      @input="handleInput"
    />
    
    <div class="validation-status mt-2">
      <el-tag type="info" class="mr-2">总计: {{ totalCount }}</el-tag>
      <el-tag type="success" class="mr-2">有效: {{ validCount }}</el-tag>
      <el-tag type="danger" v-if="invalidCount > 0">无效/重复: {{ invalidCount }}</el-tag>
    </div>

    <div v-if="invalidDomains.length > 0" class="invalid-list mt-2">
      <p class="text-danger text-sm">无效域名列表:</p>
      <ul>
        <li v-for="(item, idx) in invalidDomains" :key="idx" class="text-xs text-danger">
          {{ item.domain }}: {{ item.reason }}
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'change', 'validation'])

const rawInput = ref(props.modelValue)
const validDomains = ref([])
const invalidDomains = ref([])

const totalCount = computed(() => validDomains.value.length + invalidDomains.value.length)
const validCount = computed(() => validDomains.value.length)
const invalidCount = computed(() => invalidDomains.value.length)

watch(() => props.modelValue, (val) => {
  if (val !== rawInput.value) {
    rawInput.value = val
    validate()
  }
})

const handleInput = (val) => {
  emit('update:modelValue', val)
  emit('change', val)
  validate()
}

const validate = () => {
  const text = rawInput.value || ''
  // Split by newlines, commas, spaces, semicolons
  const parts = text.split(/[\n,;\s]+/)
  
  const valid = []
  const invalid = []
  const seen = new Set()

  for (let part of parts) {
    part = part.trim().toLowerCase()
    if (!part) continue
    
    // Remove trailing dot
    if (part.endsWith('.')) {
      part = part.slice(0, -1)
    }

    // Validation Rules
    let reason = ''
    if (part.includes('http://') || part.includes('https://')) {
      reason = '如果在包含协议(http/https)'
    } else if (part.includes('/')) {
      reason = '不能包含路径'
    } else if (part.includes(':')) {
       // Allow IPv6? Usually domains don't have : unless port.
       // Requirement says: Pure domains only.
       reason = '不能包含端口'
    } else if (!/^[a-z0-9.-]+$/.test(part)) {
       // Basic regex for domain chars
       reason = '包含非法字符'
    } else if (part.startsWith('-') || part.endsWith('-')) {
       reason = '不能以连字符开头或结尾'
    } else if (!part.includes('.')) {
       // Simple check for TLD (localhost might pass but usually rejected for public certs)
       // Let's allow localhost if needed but usually requires dot for public.
       // Requirement says "consistently validate".
       // Just strict domain validation.
       reason = '格式不正确'
    }

    if (!reason) {
      if (seen.has(part)) {
        invalid.push({ domain: part, reason: '重复' })
      } else {
        seen.add(part)
        valid.push(part)
      }
    } else {
      invalid.push({ domain: part, reason })
    }
  }

  validDomains.value = valid
  invalidDomains.value = invalid

  emit('validation', {
    valid: valid,
    invalid: invalid
  })
}
</script>

<style scoped>
.invalid-list {
  max-height: 100px;
  overflow-y: auto;
  border: 1px solid var(--el-color-danger-light-7);
  padding: 5px;
  background-color: var(--el-color-danger-light-9);
}
.text-danger { color: var(--el-color-danger); }
.text-sm { font-size: 12px; }
.text-xs { font-size: 11px; }
.mt-2 { margin-top: 8px; }
.mr-2 { margin-right: 8px; }
</style>
