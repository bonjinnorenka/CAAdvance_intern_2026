import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    watch: {
      usePolling: true,
      interval: 300,
    },
    proxy: {
      '/api': {
        target: 'http://internal-api:8080',
        changeOrigin: true,
      },
      '^/(me|report|users|user|ad_accounts)': {
        target: 'http://internal-api:8080',
        changeOrigin: true,
      },
    },
  },
})
