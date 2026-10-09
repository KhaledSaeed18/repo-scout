import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// The API and its WebSocket are proxied so the browser talks to one origin.
// Host is forwarded unchanged so the backend's same-origin WebSocket check
// sees matching Origin and Host headers.
const proxy = {
  '/api': {
    target: 'http://127.0.0.1:8080',
    ws: true,
  },
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  server: { proxy },
  preview: { proxy },
})
