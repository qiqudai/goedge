import { ref, onBeforeUnmount, watch } from 'vue'

/**
 * 通用轮询 composable：在条件满足时定期调用 fetcher，结束后自动停止。
 *
 * @param {() => Promise<boolean|void>} fetcher 返回 true 表示需要继续轮询，false/undefined 表示完成
 * @param {Object} options
 * @param {number} options.interval 轮询间隔 ms，默认 10000
 * @param {boolean} options.immediate 是否立即执行一次，默认 true
 * @param {() => boolean} options.shouldRun 决定是否应该启动轮询（如：列表中有非终态行）
 */
export function usePolling(fetcher, options = {}) {
  const {
    interval = 10000,
    immediate = true,
    shouldRun
  } = options

  const timer = ref(null)
  const running = ref(false)

  const stop = () => {
    if (timer.value) {
      clearInterval(timer.value)
      timer.value = null
    }
    running.value = false
  }

  const tick = async () => {
    if (typeof shouldRun === 'function' && !shouldRun()) {
      stop()
      return
    }
    try {
      const needContinue = await fetcher()
      if (needContinue === false) {
        stop()
      }
    } catch (e) {
      // 错误不应中断轮询；由调用方处理 toast
      console.error('[usePolling] fetcher error', e)
    }
  }

  const start = () => {
    if (running.value) return
    if (typeof shouldRun === 'function' && !shouldRun()) return
    running.value = true
    if (immediate) tick()
    timer.value = setInterval(tick, interval)
  }

  const restart = () => {
    stop()
    start()
  }

  onBeforeUnmount(stop)

  if (typeof shouldRun === 'function') {
    watch(shouldRun, (val) => {
      if (val && !running.value) start()
      else if (!val && running.value) stop()
    })
  }

  return { start, stop, restart, running }
}
