// The user guide, docs/user-guide.md, as one page per "## " section. Links into the repo point at GitHub, and a link to
// another section's heading points at the page that holds it.
import { readFileSync } from 'node:fs'
import { markdown, REPO, repo } from './site.ts'

export interface Section {
  slug: string
  title: string
  html: string
}

const ABSOLUTE_HREF = /^([a-z]+:|#|\/)/
const TITLE_LINE = /^# .*\n/
const HREF_ATTR = /href="([^"]+)"/g
const ANCHOR_HREF = /href="#([^"]+)"/g
const H2_START = /(?=<h2 id=")/
const H2_HEAD = /^<h2 id="([^"]+)">(.*?)<\/h2>/
const H2_BLOCK = /^<h2[^>]*>.*?<\/h2>\n?/
const SUBHEADING_ID = /<h[3-6] id="([^"]+)"/g

/** A link relative to docs/ points into the repo on GitHub; site and web addresses stay as they are. */
function repoLink(href: string): string {
  if (ABSOLUTE_HREF.test(href)) {
    return href
  }
  const path = new URL(href, 'https://repo.invalid/docs/').pathname.slice(1)
  return `${REPO}/blob/main/${path}`
}

let cached: Section[] | undefined

function guideHtml(): string {
  const md = readFileSync(repo('docs/user-guide.md'), 'utf8').replace(TITLE_LINE, '')
  return markdown(md).replace(HREF_ATTR, (_, href: string) => `href="${repoLink(href)}"`)
}

/** What the guide says before its first section, for the guide's front page. */
export const guideIntro = (): string => guideHtml().split(H2_START)[0] ?? ''

export function guideSections(): Section[] {
  if (cached) {
    return cached
  }
  const parts = guideHtml()
    .split(H2_START)
    .filter((p) => p.startsWith('<h2'))
  const sections = parts.map((part) => {
    const [, slug = '', title = ''] = H2_HEAD.exec(part) ?? []
    return { slug, title, html: part.replace(H2_BLOCK, '') }
  })
  // Every heading id, by the page it is on, so a "#id" link can name the page.
  const home = new Map<string, string>()
  for (const s of sections) {
    home.set(s.slug, s.slug)
    for (const [, id = ''] of s.html.matchAll(SUBHEADING_ID)) {
      home.set(id, s.slug)
    }
  }
  cached = sections.map((s) => ({
    ...s,
    html: s.html.replace(ANCHOR_HREF, (whole, id: string) => {
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
    for (const [, id = ''] of s.html.matchAll(SUBHEADING_ID)) {
      out[id] = s.slug
    }
  }
  return out
}
