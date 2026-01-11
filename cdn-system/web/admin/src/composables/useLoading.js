import { computed, ref } from 'vue'
import { ElLoading } from 'element-plus'

// Global singleton loading state.
const loadingCount = ref(0)
const loadingText = ref('正在加载中...')
const isVisible = ref(false)
const dialogCount = ref(0)
const minVisibleMs = 300
let lastShowAt = 0
let hideTimer = null
let dialogService = null
let dialogTarget = null

export function useLoading() {
  const isVisibleElement = (el) => {
    if (!el || typeof window === 'undefined') return false
    const rect = el.getBoundingClientRect()
    if (rect.width === 0 || rect.height === 0) return false
    const style = window.getComputedStyle(el)
    return style.display !== 'none' && style.visibility !== 'hidden'
  }

  const getActiveDialog = () => {
    if (typeof document === 'undefined') return null
    const dialogs = Array.from(document.querySelectorAll('.el-dialog'))
    const visible = dialogs.filter(isVisibleElement)
    if (!visible.length) return null
    return visible[visible.length - 1]
  }

  const showLoading = (text = '正在加载中...') => {
    const dialogEl = getActiveDialog()
    if (dialogEl) {
      dialogCount.value += 1
      if (dialogService && dialogTarget !== dialogEl) {
        dialogService.close()
        dialogService = null
      }
      dialogTarget = dialogEl
      if (!dialogService) {
        dialogService = ElLoading.service({
          target: dialogEl,
          text,
          background: 'rgba(255, 255, 255, 0.7)',
          lock: false
        })
      }
      return { type: 'dialog' }
    }

    loadingText.value = text
    loadingCount.value += 1
    if (hideTimer) {
      clearTimeout(hideTimer)
      hideTimer = null
    }
    if (!isVisible.value) {
      isVisible.value = true
      lastShowAt = Date.now()
    }
    return { type: 'global' }
  }

  const hideLoading = (token) => {
    if (token?.type === 'dialog') {
      dialogCount.value = Math.max(0, dialogCount.value - 1)
      if (dialogCount.value === 0 && dialogService) {
        dialogService.close()
        dialogService = null
        dialogTarget = null
      }
      return
    }

    loadingCount.value = Math.max(0, loadingCount.value - 1)
    if (loadingCount.value > 0) return
    const elapsed = Date.now() - lastShowAt
    const delay = elapsed < minVisibleMs ? minVisibleMs - elapsed : 0
    if (delay === 0) {
      isVisible.value = false
      return
    }
    hideTimer = setTimeout(() => {
      hideTimer = null
      if (loadingCount.value === 0) {
        isVisible.value = false
      }
    }, delay)
  }

  const withLoading = async (fn, text = '正在加载中...') => {
    const token = showLoading(text)
    try {
      return await fn()
    } finally {
      hideLoading(token)
    }
  }

  return {
    loading: computed(() => isVisible.value),
    loadingText,
    showLoading,
    hideLoading,
    withLoading
  }
}

