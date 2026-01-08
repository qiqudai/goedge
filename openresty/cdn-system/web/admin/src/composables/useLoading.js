import { computed, ref } from 'vue'

// Global singleton loading state.
const loadingCount = ref(0)
const loadingText = ref('正在加载中...')
const isVisible = ref(false)
const minVisibleMs = 300
let lastShowAt = 0
let hideTimer = null

export function useLoading() {
  const showLoading = (text = '正在加载中...') => {
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
  }

  const hideLoading = () => {
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
    showLoading(text)
    try {
      return await fn()
    } finally {
      hideLoading()
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

