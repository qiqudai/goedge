<template>
  <Transition name="fade">
    <div v-if="loading" class="global-loading-overlay" :style="overlayStyle">
      <div class="loading-content">
        <div class="loading-spinner">
          <div class="dot"></div>
          <div class="dot"></div>
          <div class="dot"></div>
          <div class="dot"></div>
          <div class="dot"></div>
        </div>
        <div class="loading-text">{{ loadingText }}</div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { onMounted, onUnmounted, nextTick, ref, watch } from 'vue'
import { useLoading } from '@/composables/useLoading'

const { loading, loadingText } = useLoading()
const overlayStyle = ref({})

const isVisibleElement = (el) => {
  if (!el) return false
  const rect = el.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) return false
  const style = window.getComputedStyle(el)
  return style.display !== 'none' && style.visibility !== 'hidden'
}

const findTargetRect = () => {
  const wrappers = Array.from(document.querySelectorAll('.el-dialog__wrapper'))
  const visibleWrappers = wrappers.filter(isVisibleElement)
  const activeWrapper = visibleWrappers[visibleWrappers.length - 1]
  if (activeWrapper) {
    const dialog = activeWrapper.querySelector('.el-dialog')
    const rect = (dialog || activeWrapper).getBoundingClientRect()
    return {
      top: rect.top,
      left: rect.left,
      width: rect.width,
      height: rect.height
    }
  }

  const main = document.querySelector('.el-main')
  if (!main) {
    return { top: 0, left: 0, width: window.innerWidth, height: window.innerHeight }
  }
  const tabContents = Array.from(main.querySelectorAll('.el-tabs__content'))
  const activeTabContent = tabContents.find(isVisibleElement)
  const target = activeTabContent || main
  const rect = target.getBoundingClientRect()
  return {
    top: rect.top,
    left: rect.left,
    width: rect.width,
    height: rect.height
  }
}

const updateOverlayStyle = () => {
  const rect = findTargetRect()
  overlayStyle.value = {
    top: `${rect.top}px`,
    left: `${rect.left}px`,
    width: `${rect.width}px`,
    height: `${rect.height}px`
  }
}

const handleResize = () => {
  if (!loading.value) return
  updateOverlayStyle()
}

watch(
  () => loading.value,
  async (val) => {
    if (!val) return
    await nextTick()
    updateOverlayStyle()
  }
)

onMounted(() => {
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.global-loading-overlay {
  position: fixed;
  background: rgba(255, 255, 255, 0.7);
  backdrop-filter: blur(8px);
  z-index: 99999;
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: column;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.loading-text {
  font-size: 16px;
  color: #409eff;
  font-weight: 500;
  letter-spacing: 1px;
}

/* Premium Spinner */
.loading-spinner {
  width: 50px;
  height: 50px;
  position: relative;
}

.dot {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  animation: rotate 2.5s infinite linear both;
}

.dot::before {
  content: '';
  display: block;
  width: 25%;
  height: 25%;
  background-color: #409eff;
  border-radius: 100%;
  animation: dotBefore 2s infinite ease-in-out both;
}

.dot:nth-child(1) { animation-delay: -1.1s; }
.dot:nth-child(1)::before { animation-delay: -1.1s; }
.dot:nth-child(2) { animation-delay: -1.0s; }
.dot:nth-child(2)::before { animation-delay: -1.0s; }
.dot:nth-child(3) { animation-delay: -0.9s; }
.dot:nth-child(3)::before { animation-delay: -0.9s; }
.dot:nth-child(4) { animation-delay: -0.8s; }
.dot:nth-child(4)::before { animation-delay: -0.8s; }
.dot:nth-child(5) { animation-delay: -0.7s; }
.dot:nth-child(5)::before { animation-delay: -0.7s; }

@keyframes rotate {
  100% { transform: rotate(360deg); }
}

@keyframes dotBefore {
  50% { transform: scale(0.4); opacity: 0.3; }
  100% { transform: scale(1); opacity: 1; }
}

/* Fade Transition */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
