import { ref, computed } from 'vue'

// 全局单例状态
const loadingCount = ref(0)
const loadingText = ref('正在加载中...')
const isVisible = computed(() => loadingCount.value > 0)

/**
 * 全局 Loading 状态管理
 */
export function useLoading() {
    /**
     * 开启加锁 Loading
     * @param {string} text - 显示的文字内容
     */
    const showLoading = (text = '正在加载中...') => {
        loadingText.value = text
        loadingCount.value++
    }

    /**
     * 关闭 Loading
     */
    const hideLoading = () => {
        loadingCount.value = Math.max(0, loadingCount.value - 1)
    }

    /**
     * 自动管理异步任务的 Loading
     * @param {Function} fn - 异步函数
     * @param {string} text - 加载中文字
     */
    const withLoading = async (fn, text = '正在加载中...') => {
        showLoading(text)
        try {
            return await fn()
        } finally {
            hideLoading()
        }
    }

    return {
        loading: isVisible,
        loadingText,
        showLoading,
        hideLoading,
        withLoading
    }
}
