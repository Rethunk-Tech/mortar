// Every release's player-facing notes (build/release/notes/v*.md), newest first, dated from the AppStream metainfo's
// release list so no git history is needed to build the changelog, the news posts or the feeds.
import { readdirSync, readFileSync } from 'node:fs'
import { escapeHTML, repo } from './site.ts'

const NOTES = repo('build/release/notes')
const METAINFO = repo('build/linux/tech.rethunk.Mortar.metainfo.xml')
const NOTES_HEADING = /^(##|###) (.+)$/
const INLINE_CODE = /`([^`]+)`/g
const RELEASE_VERSION = /<release version="(\d+\.\d+\.\d+)" date=/g
const NOTES_FILE = /^v(\d+\.\d+\.\d+)\.md$/

function byVersionDesc(a: string, b: string): number {
  const pa = a.split('.').map(Number)
  const pb = b.split('.').map(Number)
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const d = (pb[i] ?? 0) - (pa[i] ?? 0)
    if (d !== 0) {
      return d
    }
  }
  return 0
}

/** A notes file's "## Group", "### Area" and "- line" lines as HTML; New shows in full, the other groups fold behind
 * their name and count. */
function notesHtml(md: string): string {
  const groups: { name: string; lines: string[]; count: number }[] = []
  let open = false
  const close = () => {
    const g = groups.at(-1)
    if (open && g) {
      g.lines.push('</ul>')
      open = false
    }
  }
  for (const line of md.split('\n')) {
    const heading = NOTES_HEADING.exec(line)
    if (heading?.[1] === '##') {
      close()
      groups.push({ name: heading[2] ?? '', lines: [], count: 0 })
      continue
    }
    if (groups.length === 0) {
      groups.push({ name: '', lines: [], count: 0 })
    }
    const g = groups.at(-1)
    if (!g) {
      continue
    }
    if (heading) {
      close()
      g.lines.push(`<h4>${escapeHTML(heading[2] ?? '')}</h4>`)
    } else if (line.startsWith('- ')) {
      if (!open) {
        g.lines.push('<ul>')
        open = true
      }
      g.count++
      g.lines.push(`<li>${escapeHTML(line.slice(2)).replace(INLINE_CODE, '<code>$1</code>')}</li>`)
    }
  }
  close()
  const out = groups.flatMap((g) => {
    if (g.name === '' || g.name === 'New') {
      return [...(g.name ? [`<h3>${escapeHTML(g.name)}</h3>`] : []), ...g.lines]
    }
    return [
      `<details><summary><h3>${escapeHTML(g.name)} (${g.count})</h3></summary>`,
      ...g.lines,
      '</details>',
    ]
  })
  return out.map((l) => `        ${l}`).join('\n')
}

export interface Release {
  version: string
  date: string
  md: string
  html: string
}

/** Every release with written notes, newest first, dated from the metainfo. Notes drafted for the next release, newer than any dated one, wait until it is cut. */
export function releases(): Release[] {
  const metainfo = readFileSync(METAINFO, 'utf8')
  const dated = [...metainfo.matchAll(RELEASE_VERSION)].map((m) => m[1] ?? '')
  const newest = dated.sort(byVersionDesc)[0] ?? ''
  return readdirSync(NOTES)
    .map((f) => NOTES_FILE.exec(f)?.[1])
    .filter((v): v is string => v !== undefined)
    .sort(byVersionDesc)
    .filter((v) => newest === '' || byVersionDesc(v, newest) >= 0)
    .map((version) => {
      const date = new RegExp(
        `<release version="${version.replaceAll('.', '\\.')}" date="([^"]+)"`,
      ).exec(metainfo)?.[1]
      if (!date) {
        throw new Error(`site: ${METAINFO} has no dated <release version="${version}">`)
      }
      const md = readFileSync(`${NOTES}/v${version}.md`, 'utf8')
      return { version, date, md, html: notesHtml(md) }
    })
}

/** How many changes a release's notes list: one per "- " line. */
export const changeCount = (r: Release) => r.md.split('\n').filter((l) => l.startsWith('- ')).length
