import { SITE } from '../lib/site.ts'

export const GET = () =>
  new Response(`User-agent: *\nAllow: /\n\nSitemap: ${SITE}/sitemap-index.xml\n`)
