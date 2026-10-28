// Every release's player-facing notes (build/release/notes/v*.md), newest first, dated from the AppStream metainfo's
// release list so no git history is needed to build the changelog, the news posts or the feeds.
import { readdirSync, readFileSync } from 'node:fs'
import { escapeHTML, markdown, repo } from './site.ts'

const NOTES = repo('build/release/notes')
const METAINFO = repo('build/linux/tech.rethunk.Mortar.metainfo.xml')
const GROUP_START = /^## /m
const AREA_HEADING = /^### /gm
const HEADING_ID = /(<h4) id="[^"]*"/g
const LIST_ITEM = /^- /gm
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

/** A notes file as HTML, one block per "## Group"; New shows in full, the other groups fold behind their name and
 * count. */
function notesHtml(md: string): string {
  return md
    .split(GROUP_START)
    .flatMap((part, i) => {
      const [first = '', ...rest] = part.split('\n')
      const name = i === 0 ? '' : first.trim()
      const body = i === 0 ? part : rest.join('\n')
      // Several releases share a page, so the area headings must not carry the ids markdown() gives them.
      const html = markdown(body.replace(AREA_HEADING, '#### ')).replace(HEADING_ID, '$1')
      if (name === '') {
        return html ? [html] : []
      }
      if (name === 'New') {
        return [`<h3>${escapeHTML(name)}</h3>`, html]
      }
      const count = body.match(LIST_ITEM)?.length ?? 0
      return [
        `<details><summary><h3>${escapeHTML(name)} (${count})</h3></summary>`,
        html,
        '</details>',
      ]
    })
    .join('\n')
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
