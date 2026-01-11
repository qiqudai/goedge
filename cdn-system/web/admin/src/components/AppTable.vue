<template>
  <div class="app-table">
  <el-table
    ref="elTableRef"
    :data="tableData"
    v-loading="loading"
    v-bind="tableAttrs"
    @header-dragend="handleHeaderDragend"
    @selection-change="handleSelectionChange"
    :row-key="resolvedRowKey"
  >
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
import { computed, ref, useAttrs, onMounted, nextTick, watch, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useTablePersistence } from '@/utils/tablePersistence'
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
  paginationClass: { type: String, default: 'pagination-container' },
  storageKey: { type: String, default: '' }
})

const emit = defineEmits([
  'update:current-page',
  'update:page-size',
  'size-change',
  'current-change'
])

const attrs = useAttrs()
const elTableRef = ref(null)
const route = useRoute()

const tableAttrs = computed(() => {
  const { onSelectionChange, ...rest } = attrs
  return rest
})
const resolvedRowKey = computed(() => attrs.rowKey || attrs['row-key'] || 'id')

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

const selectionStorageKey = computed(() => {
  const base = props.persistKey || 'default'
  return `table-selection:${route.path}:${base}`
})

const isRestoringSelection = ref(false)

const getRowKeyValue = (row) => {
  const key = resolvedRowKey.value
  if (typeof key === 'function') {
    return key(row)
  }
  return row ? row[key] : undefined
}

const saveSelection = (rows) => {
  const key = selectionStorageKey.value
  if (!key) return
  try {
    const ids = rows.map(getRowKeyValue).filter(v => v !== undefined && v !== null)
    sessionStorage.setItem(key, JSON.stringify(ids))
  } catch (e) {
    console.error('Failed to save selection', e)
  }
}

const restoreSelection = () => {
  const key = selectionStorageKey.value
  if (!key) return
  let saved = []
  try {
    saved = JSON.parse(sessionStorage.getItem(key) || '[]')
  } catch {
    saved = []
  }
  if (!saved.length) return
  const table = elTableRef.value
  if (!table) return
  const currentKeys = new Set()
  isRestoringSelection.value = true
  table.clearSelection()
  tableData.value.forEach(row => {
    const rowKey = getRowKeyValue(row)
    if (saved.includes(rowKey)) {
      table.toggleRowSelection(row, true)
      currentKeys.add(rowKey)
    }
  })
  isRestoringSelection.value = false
  try {
    sessionStorage.setItem(key, JSON.stringify(Array.from(currentKeys)))
  } catch (e) {
    console.error('Failed to update selection', e)
  }
}

const handleSelectionChange = (rows) => {
  if (!isRestoringSelection.value) {
    saveSelection(rows || [])
  }
  if (typeof attrs.onSelectionChange === 'function') {
    attrs.onSelectionChange(rows)
  }
}

const handleSizeChange = (size) => {
  if (!hasExternalPaging.value) {
    innerCurrentPage.value = 1
  }
  emit('size-change', size)
}

const handleCurrentChange = (page) => {
  emit('current-change', page)
}

const columnStorageKey = computed(() => props.storageKey || props.persistKey || route.path)

// Column Persistence
const { handleHeaderDragend } = useTablePersistence(
  columnStorageKey,
  elTableRef
)

watch(
  () => tableData.value,
  () => {
    nextTick(() => {
      restoreSelection()
    })
  },
  { deep: true }
)

onMounted(() => {
  nextTick(() => {
    restoreSelection()
  })
})

onBeforeUnmount(() => {
  const key = selectionStorageKey.value
  if (key) {
    sessionStorage.removeItem(key)
  }
})
</script>
