import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// API proxy target for development
const API_PROXY_TARGET = 'http://localhost:5000'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor': ['vue', 'vue-router', 'pinia'],
          'charts': ['highcharts']
        }
      }
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: API_PROXY_TARGET,
        changeOrigin: true
      },
      '/auth': {
        target: API_PROXY_TARGET,
        changeOrigin: true
      },
      '/health': {
        target: API_PROXY_TARGET,
        changeOrigin: true
      }
    }
  }
})
