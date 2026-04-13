import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    watch: {
      usePolling: true,
      interval: 1000,
    },
    proxy: {
      '/api/v1': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
