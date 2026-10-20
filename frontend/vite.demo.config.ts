import { readFileSync } from 'node:fs'
import { mergeConfig } from 'vite'
import app from './vite.config.ts'

// The website's offline demo (scripts/site-demo.ts): the real app booted on recorded backend answers, with relative
// asset paths so it works from whatever folder the site serves it under.
export default mergeConfig(app, {
  base: './',
  plugins: [
    {
      name: 'mortar-demo',
      transformIndexHtml: {
        order: 'pre',
        handler: (html: string) => html.replace('/src/main.tsx', '/src/demo/entry.ts'),
      },
      // The backdrop the app asks its backend for, as the bundled default wallpaper.
      generateBundle() {
        this.emitFile({
          type: 'asset',
          fileName: 'backdrop',
          source: readFileSync(new URL('../internal/backdrop/f44-01-night.jpg', import.meta.url)),
        })
      },
    },
  ],
})
