import sitemap from '@astrojs/sitemap'
import { defineConfig } from 'astro/config'
import { og } from './scripts/og.ts'

// Pages that are not for search: a shared profile, the Discord redirects and the 404.
const unlisted = /\/(p|s(\/[^/]+){0,2}|go(\/[^/]+)?|404)\/?$/

export default defineConfig({
  site: 'https://mortar.rethunk.tech',
  outDir: '../site',
  build: { format: 'directory' },
  // The guide page was renamed once more sources joined it; old links still land on it.
  redirects: { '/guide/modrinth-and-itchio': '/guide/modrinth-modio-paradox-mods-and-itchio' },
  // Astro's compressor drops the line break between a word and an inline tag ("run<code>"), gluing them together.
  compressHTML: false,
  devToolbar: { enabled: false },
  integrations: [og(), sitemap({ filter: (page) => !unlisted.test(new URL(page).pathname) })],
})
