import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('/zrender/')) return 'vendor-zrender'
          if (id.includes('/echarts/')) return 'vendor-echarts'
          if (id.includes('/element-plus/') || id.includes('@element-plus')) return 'vendor-element-plus'
          if (id.includes('/vue') || id.includes('/pinia/')) return 'vendor-vue'
          return 'vendor'
        }
      }
    }
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@cdn-common': fileURLToPath(new URL('../../common', import.meta.url))
    }
  },
  server: {
    host: '127.0.0.1',
    port: 5176
  }
})
