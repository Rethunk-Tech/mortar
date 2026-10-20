import sitemap from '@astrojs/sitemap'
import { defineConfig } from 'astro/config'

// Pages that are not for search: a shared profile, the Discord redirects and the 404.
const unlisted = /\/(p|go(\/[^/]+)?|404)\/?$/

export default defineConfig({
  site: 'https://mortar.rethunk.tech',
  outDir: '../site',
  build: { format: 'directory' },
  devToolbar: { enabled: false },
  integrations: [sitemap({ filter: (page) => !unlisted.test(new URL(page).pathname) })],
})
