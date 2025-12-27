<template>
  <div class="app-table">
    <el-table :data="tableData" v-loading="loading" v-bind="tableAttrs">
      <slot />
    </el-table>
    <div v-if="showPaginationComputed" :class="paginationClass">
      <AppPagination
        v-model:current-page="currentPageProxy"
        v-model:page-size="pageSizeProxy"
        :total="totalComputed"
        :page-sizes="pageSizes"
        :layout="layout"
        :persist-key="persistKey"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
</template>

<script>
export default {
  inheritAttrs: false
}
</script>

<script setup>
import { computed, ref, useAttrs } from 'vue'
import AppPagination from './AppPagination.vue'

const props = defineProps({
  data: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  currentPage: { type: [Number, String], default: undefined },
  pageSize: { type: [Number, String], default: undefined },
  total: { type: [Number, String], default: undefined },
  pageSizes: { type: Array, default: () => [10, 30, 50, 100, 200, 300, 500] },
  layout: { type: String, default: 'total, sizes, prev, pager, next, jumper' },
  persistKey: { type: String, default: 'default' },
  showPagination: { type: Boolean, default: true },
  paginationClass: { type: String, default: 'pagination-container' }
})

const emit = defineEmits([
  'update:current-page',
  'update:page-size',
  'size-change',
  'current-change'
])

const attrs = useAttrs()

const tableAttrs = computed(() => ({ ...attrs }))

const hasExternalPaging = computed(
  () => props.currentPage !== undefined && props.pageSize !== undefined
)

const innerCurrentPage = ref(1)
const innerPageSize = ref(props.pageSizes[0] || 10)

const currentPageProxy = computed({
  get: () => (hasExternalPaging.value ? Number(props.currentPage) : innerCurrentPage.value),
  set: (val) => {
    if (hasExternalPaging.value) {
      emit('update:current-page', val)
    } else {
      innerCurrentPage.value = Number(val) || 1
    }
  }
})

const pageSizeProxy = computed({
  get: () => (hasExternalPaging.value ? Number(props.pageSize) : innerPageSize.value),
  set: (val) => {
    if (hasExternalPaging.value) {
      emit('update:page-size', val)
    } else {
      innerPageSize.value = Number(val) || innerPageSize.value
    }
  }
})

const totalComputed = computed(() => {
  const parsed = Number(props.total)
  if (!Number.isNaN(parsed)) {
    return parsed
  }
  return Array.isArray(props.data) ? props.data.length : 0
})

const tableData = computed(() => {
  const base = Array.isArray(props.data) ? props.data : []
  if (hasExternalPaging.value) {
    return base
  }
  const size = Number(pageSizeProxy.value) || base.length || 0
  const page = Number(currentPageProxy.value) || 1
  const start = (page - 1) * size
  return base.slice(start, start + size)
})

const showPaginationComputed = computed(() => {
  if (!props.showPagination) {
    return false
  }
  return (
    Number.isFinite(Number(totalComputed.value)) &&
    Number.isFinite(Number(currentPageProxy.value)) &&
    Number.isFinite(Number(pageSizeProxy.value))
  )
})

const handleSizeChange = (size) => {
  if (!hasExternalPaging.value) {
    innerCurrentPage.value = 1
  }
  emit('size-change', size)
}

const handleCurrentChange = (page) => {
  emit('current-change', page)
}
</script>
