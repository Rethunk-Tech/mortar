import { lingui } from '@lingui/vite-plugin'
import react from '@vitejs/plugin-react'
import wails from '@wailsio/runtime/plugins/vite'
import { defineConfig } from 'vite'

// https://vitejs.dev/config/
const DEFAULT_PORT = 9245

export default defineConfig({
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || DEFAULT_PORT,
    strictPort: true,
  },
  plugins: [react(), lingui({ macroTransform: true }), wails('./bindings')],
})
