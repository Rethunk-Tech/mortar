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
  build: {
    rolldownOptions: {
      output: {
        // Libraries change less often than the app, and one bundle of everything passed Vite's 500 kB chunk warning.
        codeSplitting: {
          groups: [
            { name: 'mui', test: /node_modules[\\/]@(mui|emotion|popperjs)[\\/]/ },
            { name: 'react', test: /node_modules[\\/](react|react-dom|scheduler)[\\/]/ },
            { name: 'vendor', test: /node_modules[\\/]/ },
          ],
        },
      },
    },
  },
})
