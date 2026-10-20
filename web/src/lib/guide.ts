// The user guide, docs/user-guide.md, as one page per "## " section. Links into the repo point at GitHub, and a link to
// another section's heading points at the page that holds it.
import { readFileSync } from 'node:fs'
import { markdown, REPO, repo } from './site.ts'

export interface Section {
  slug: string
  title: string
  html: string
}

/** A link relative to docs/ points into the repo on GitHub; site and web addresses stay as they are. */
function repoLink(href: string): string {
  if (/^([a-z]+:|#|\/)/.test(href)) {
    return href
  }
  const path = new URL(href, 'https://repo.invalid/docs/').pathname.slice(1)
  return `${REPO}/blob/main/${path}`
}

let cached: Section[] | undefined

export function guideSections(): Section[] {
  if (cached) {
    return cached
  }
  const md = readFileSync(repo('docs/user-guide.md'), 'utf8').replace(/^# .*\n/, '')
  const html = markdown(md).replace(
    /href="([^"]+)"/g,
    (_, href: string) => `href="${repoLink(href)}"`,
  )
  const parts = html.split(/(?=<h2 id=")/).filter((p) => p.startsWith('<h2'))
  const sections = parts.map((part) => {
    const [, slug = '', title = ''] = /^<h2 id="([^"]+)">(.*?)<\/h2>/.exec(part) ?? []
    return { slug, title, html: part.replace(/^<h2[^>]*>.*?<\/h2>\n?/, '') }
  })
  // Every heading id, by the page it is on, so a "#id" link can name the page.
  const home = new Map<string, string>()
  for (const s of sections) {
    home.set(s.slug, s.slug)
    for (const [, id = ''] of s.html.matchAll(/<h[3-6] id="([^"]+)"/g)) {
      home.set(id, s.slug)
    }
  }
  cached = sections.map((s) => ({
    ...s,
    html: s.html.replace(/href="#([^"]+)"/g, (whole, id: string) => {
      const page = home.get(id)
      if (page === undefined) {
        return whole
      }
      return page === id ? `href="/guide/${page}/"` : `href="/guide/${page}/#${id}"`
    }),
  }))
  return cached
}

/** Each heading id on the guide's pages, by the page that holds it: /guide/#id from the one-page guide still lands. */
export function guideAnchors(): Record<string, string> {
  const out: Record<string, string> = {}
  for (const s of guideSections()) {
    out[s.slug] = s.slug
    for (const [, id = ''] of s.html.matchAll(/<h[3-6] id="([^"]+)"/g)) {
      out[id] = s.slug
    }
  }
  return out
}
