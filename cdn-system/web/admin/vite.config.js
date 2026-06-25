import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// https://vitejs.dev/config/
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
      '@': path.resolve(__dirname, 'src'),
      '@cdn-common': path.resolve(__dirname, '../../common')
    }
  },
  server: {
    proxy: {
      '/api': {
        target: 'https://goai.665305.cc',
        changeOrigin: true,
        rewrite: (path) => path
      }
    }
  }
})
