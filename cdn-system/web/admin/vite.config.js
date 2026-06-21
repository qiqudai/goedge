import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
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
