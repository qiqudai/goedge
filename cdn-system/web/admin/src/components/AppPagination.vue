<template>
  <el-pagination
    v-bind="attrs"
    v-model:current-page="currentPageProxy"
    v-model:page-size="pageSizeProxy"
    :total="total"
    :page-sizes="pageSizes"
    :layout="layout"
    @size-change="handleSizeChange"
    @current-change="handleCurrentChange"
  />
</template>

<script setup>
import { computed, onMounted, watch, useAttrs } from 'vue'
import { useRoute } from 'vue-router'
import { buildPageSizeKey, loadPageSize, savePageSize } from '@/utils/pagination'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  currentPage: { type: Number, default: 1 },
  pageSize: { type: Number, default: 10 },
  total: { type: Number, default: 0 },
  pageSizes: { type: Array, default: () => [10, 30, 50, 100, 200, 300, 500] },
  layout: { type: String, default: 'total, sizes, prev, pager, next, jumper' },
  persistKey: { type: String, default: 'default' }
})

const emit = defineEmits([
  'update:current-page',
  'update:page-size',
  'size-change',
  'current-change'
])

const attrs = useAttrs()
const route = useRoute()

const currentPageProxy = computed({
  get: () => props.currentPage,
  set: (val) => emit('update:current-page', val)
})

const pageSizeProxy = computed({
  get: () => props.pageSize,
  set: (val) => emit('update:page-size', val)
})

const storageKey = computed(() => buildPageSizeKey(route.path, props.persistKey))

onMounted(() => {
  const base = props.pageSize || props.pageSizes[0] || 10
  const saved = loadPageSize(storageKey.value, base)
  if (saved !== props.pageSize) {
    emit('update:page-size', saved)
  }
})

watch(
  () => pageSizeProxy.value,
  (size) => {
    savePageSize(storageKey.value, size)
  }
)

const handleSizeChange = (size) => {
  emit('size-change', size)
}

const handleCurrentChange = (page) => {
  emit('current-change', page)
}
</script>

