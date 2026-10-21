import sitemap from '@astrojs/sitemap'
import { defineConfig } from 'astro/config'
import { og } from './scripts/og.ts'

// Pages that are not for search: a shared profile, the Discord redirects and the 404.
const unlisted = /\/(p|go(\/[^/]+)?|404)\/?$/

export default defineConfig({
  site: 'https://mortar.rethunk.tech',
  outDir: '../site',
  build: { format: 'directory' },
  // Astro's compressor drops the line break between a word and an inline tag ("run<code>"), gluing them together.
  compressHTML: false,
  devToolbar: { enabled: false },
  integrations: [og(), sitemap({ filter: (page) => !unlisted.test(new URL(page).pathname) })],
})
