import { onMounted, nextTick, unref } from 'vue'

export function useTablePersistence(storageKey, tableRef) {
  const getStorageKey = () => {
    const key = unref(storageKey)
    if (!key) return ''
    return `table-cols-${key}`
  }

  const getColumnKey = (column) => {
    if (!column) return ''
    return column.property || column.columnKey || column.label || column.type || column.id || ''
  }

  const handleHeaderDragend = (newWidth, oldWidth, column) => {
    const keyStr = getStorageKey()
    const colKey = getColumnKey(column)
    if (!keyStr || !colKey) return
  
    try {
      const saved = localStorage.getItem(keyStr)
      const cols = saved ? JSON.parse(saved) : {}
      cols[colKey] = newWidth
      localStorage.setItem(keyStr, JSON.stringify(cols))
    } catch (e) {
      console.error('Failed to save column width', e)
    }
  }

  const restoreColumnWidths = () => {
    const keyStr = getStorageKey()
    if (!keyStr) return
    
    nextTick(() => {
      try {
        const saved = localStorage.getItem(keyStr)
        if (!saved) return
        
        const cols = JSON.parse(saved)
        const table = unref(tableRef)
        
        if (table && table.columns) {
          let hasChanges = false
          table.columns.forEach(col => {
            const colKey = getColumnKey(col)
            if (colKey && cols[colKey]) {
              col.width = cols[colKey]
              col.realWidth = cols[colKey]
              hasChanges = true
            }
          })
          if (hasChanges) table.doLayout()
        }
      } catch (e) {
        console.error('Failed to restore column widths', e)
      }
    })
  }

  onMounted(() => {
    restoreColumnWidths()
  })

  return {
    handleHeaderDragend,
    restoreColumnWidths
  }
}
