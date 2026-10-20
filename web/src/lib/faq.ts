// The FAQ's questions (web/src/content/faq.md), the "Why Mortar?" comparisons (web/src/content/why/<manager>.md) and the
// outside facts they quote.
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { brandIcon } from './brands.ts'
import { modSites, SOURCE_NAMES } from './games.ts'
import { markdown, repo } from './site.ts'

const FAQ = repo('web/src/content/faq.md')
const WHY = repo('web/src/content/why')

const WHY_HEADING = /^# Why Mortar\? \(vs (.+)\)\n/
const MD_EXT = /\.md$/
const TITLE_LINE = /^# .*\n/
const DOC_TITLE = /^# (.+)\n/
const QUESTION_SPLIT = /^## /m
const HEADING_ID = /id="([^"]+)"/
const MD_LINK = /\[([^\]]+)\]\([^)]+\)/g
const WHITESPACE = /\s+/g
const SOURCE_TOKEN_RUN = /(?::[a-z]+:\s*)+/g
const TOKEN_NAME = /[a-z]+/g
const CONFIG_VERSION = /^\s+version: "([^"]+)"/m
const GAMES_ROW = /^\| Games \| ([^|]*)\| ([^|]*)\|$/m
const FIRST_NUMBER = /\d+/
const OVER_PREFIX = /^over /i
const TILE_COMMENT = /<!-- tile: (.+?) -->/
const RIVAL_TOKEN = /\b[A-Z0-9]+_(?:COUNT|RELEASED?)\b/g
const TABLE_ROW = /<tr><td>(.*?)<\/td>(.*?)<\/tr>/g

export interface Why {
  id: string
  /** The other manager's name, from the page's first heading "# Why Mortar? (vs <name>)". */
  name: string
  md: string
}

export function whyPages(): Why[] {
  if (!existsSync(WHY)) {
    return []
  }
  return readdirSync(WHY)
    .filter((f) => f.endsWith('.md'))
    .sort()
    .map((f) => {
      const md = readFileSync(`${WHY}/${f}`, 'utf8')
      const name = WHY_HEADING.exec(md)?.[1]
      if (!name) {
        throw new Error(`site: ${WHY}/${f} must start with "# Why Mortar? (vs <manager>)"`)
      }
      return { id: f.replace(MD_EXT, ''), name, md: md.replace(TITLE_LINE, '') }
    })
}

/** The FAQ's questions and their Markdown answers, in file order. */
export function questions(gameCount: number, whys: Why[]): { q: string; a: string; id: string }[] {
  const whyLinks = whys.map((w) => `- [Why Mortar? (vs ${w.name})](/why/${w.id}/)`).join('\n')
  const md = readFileSync(FAQ, 'utf8')
    .replaceAll('GAME_COUNT', String(gameCount))
    .replaceAll('WHY_LINKS', whyLinks)
  return md
    .split(QUESTION_SPLIT)
    .filter((part) => part.trim() !== '')
    .map((part) => {
      const [q = '', ...rest] = part.split('\n')
      const id =
        Bun.markdown.html(`## ${q}`, { headings: { ids: true } }).match(HEADING_ID)?.[1] ?? ''
      return { q: q.trim(), a: rest.join('\n').trim(), id }
    })
}

export const plain = (md: string) => md.replace(MD_LINK, '$1').replace(WHITESPACE, ' ').trim()

/** A plain Markdown document, titled by its first heading. */
export function doc(path: string): { title: string; html: string } {
  const md = readFileSync(repo(path), 'utf8')
  const title = DOC_TITLE.exec(md)?.[1]
  if (!title) {
    throw new Error(`site: ${path} must start with "# <title>"`)
  }
  return { title, html: markdown(md.replace(TITLE_LINE, '')) }
}

/** A run of `:source:` tokens (`:nexus: :github:`) as one row of mod-site icons. */
const sourceIcons = (html: string) =>
  html.replace(SOURCE_TOKEN_RUN, (run) => {
    const ids = run.match(TOKEN_NAME) ?? []
    if (!ids.every((id) => id in SOURCE_NAMES || id === 'hexium')) {
      return run
    }
    const icons = ids.map((id) => brandIcon(id, SOURCE_NAMES[id] ?? 'Hexium')).join('')
    return `<span class="icon-row">${icons}</span>${run.endsWith(' ') ? ' ' : ''}`
  })

// Facts about other managers read from their own repositories by `bun run --cwd web refresh` (which the release's
// site publish runs too) and committed, so building the site stays offline. A page names a game count as
// R2MODMAN_COUNT, a latest release as MO2_RELEASE and its month as MO2_RELEASED.
const RIVALS = repo('web/src/data/rival-facts.json')
/** The number `re` captures in `body`; a source reworded past `re` fails rather than writing a wrong figure. */
function captured(body: string, re: RegExp): number {
  const n = Number(re.exec(body)?.[1])
  if (!Number.isInteger(n)) {
    throw new Error(`no match for ${re}`)
  }
  return n
}

/** How many items `re` finds in the part of `body` from `start` to the first `end` after it. */
function itemsBetween(body: string, start: string, end: string, re: RegExp): number {
  const from = body.indexOf(start)
  const to = body.indexOf(end, from + start.length)
  const n = from < 0 || to < 0 ? 0 : (body.slice(from, to).match(re) ?? []).length
  if (n === 0) {
    throw new Error(`no items between "${start}" and "${end}"`)
  }
  return n
}

const MO2_GAMES =
  'https://raw.githubusercontent.com/wiki/ModOrganizer2/modorganizer/Game-Support.md'
const CURSEFORGE_GAMES =
  'https://support.curseforge.com/support/solutions/articles/9000193488-getting-started'
const BULLET = /^\s*[*-] /gm
const VORTEX_GAMES = /support for over (\d+) different games/
const HTML_ITEM = /<li\b/g

const RIVAL_COUNTS: Record<string, { url: string; count: (body: string) => number }> = {
  r2modman: {
    url: 'https://raw.githubusercontent.com/ebkr/r2modmanPlus/develop/src/assets/data/ecosystem.json',
    count: (body) =>
      Object.values(
        (JSON.parse(body) as { games: Record<string, { r2modman?: unknown }> }).games,
      ).filter((g) => g.r2modman).length,
  },
  gale: {
    url: 'https://raw.githubusercontent.com/Kesomannen/gale/master/src-tauri/games.json',
    count: (body) => (JSON.parse(body) as unknown[]).length,
  },
  vortex: {
    url: 'https://raw.githubusercontent.com/Nexus-Mods/Vortex/master/README.md',
    count: (body) => captured(body, VORTEX_GAMES),
  },
  mo2full: {
    url: MO2_GAMES,
    count: (body) => itemsBetween(body, 'are [fully]', 'have [basic support]', BULLET),
  },
  mo2basic: {
    url: MO2_GAMES,
    count: (body) => itemsBetween(body, 'have [basic support]', '###', BULLET),
  },
  cfoverwolf: {
    url: CURSEFORGE_GAMES,
    count: (body) =>
      itemsBetween(body, '<strong>Overwolf CurseForge App</strong><ul>', '</ul>', HTML_ITEM),
  },
  cfstandalone: {
    url: CURSEFORGE_GAMES,
    count: (body) => itemsBetween(body, '<strong>Standalone App</strong><ul>', '</ul>', HTML_ITEM),
  },
  cflinux: {
    url: CURSEFORGE_GAMES,
    count: (body) => itemsBetween(body, 'id="Linux"', '</ul>', HTML_ITEM),
  },
}
const RIVAL_RELEASES: Record<string, string> = {
  mo2: 'ModOrganizer2/modorganizer',
  vortex: 'Nexus-Mods/Vortex',
  stardrop: 'Floogen/Stardrop',
}

async function getText(url: string): Promise<string> {
  const token = process.env.GITHUB_TOKEN
  const res = await fetch(
    url,
    token && url.startsWith('https://api.github.com/')
      ? { headers: { Authorization: `Bearer ${token}` } }
      : {},
  )
  if (!res.ok) {
    throw new Error(`${url} answered ${res.status}`)
  }
  return res.text()
}

// A source that cannot be fetched keeps its committed figure, so one site being down never blocks a release's
// publish; the warning names it for the next refresh. A source that answers but no longer parses (a reworded page)
// fails the refresh, because keeping the old figure would present it as freshly checked.
export async function refreshRivalFacts(): Promise<void> {
  const prev = rivalFacts()
  const out: Record<string, number | string> = {}
  const keep = (keys: string[], url: string, err: unknown) => {
    for (const k of keys) {
      if (prev[k] === undefined) {
        throw new Error(`site: ${k} from ${url}: ${String(err)}`)
      }
      out[k] = prev[k]
    }
    process.stderr.write(`site: kept ${keys.join(', ')}: ${url}: ${String(err)}\n`)
  }
  for (const [id, src] of Object.entries(RIVAL_COUNTS)) {
    let body: string
    try {
      body = await getText(src.url)
    } catch (err) {
      keep([`${id}_count`], src.url, err)
      continue
    }
    try {
      out[`${id}_count`] = src.count(body)
    } catch (err) {
      throw new Error(`site: ${id}_count no longer parses from ${src.url}: ${String(err)}`)
    }
  }
  for (const [id, slug] of Object.entries(RIVAL_RELEASES)) {
    const url = `https://api.github.com/repos/${slug}/releases/latest`
    try {
      const rel = JSON.parse(await getText(url)) as { tag_name: string; published_at: string }
      out[`${id}_release`] = rel.tag_name
      out[`${id}_released`] = new Date(rel.published_at).toLocaleDateString('en-US', {
        month: 'long',
        year: 'numeric',
        timeZone: 'UTC',
      })
    } catch (err) {
      keep([`${id}_release`, `${id}_released`], url, err)
    }
  }
  writeFileSync(RIVALS, `${JSON.stringify(out, null, 2)}\n`)
}

export const rivalFacts = (): Record<string, number | string> =>
  JSON.parse(readFileSync(RIVALS, 'utf8')) as Record<string, number | string>

/** The app version, from build/config.yml where it is set; a comparison page names it as MORTAR_VERSION. */
export const appVersion = (): string =>
  CONFIG_VERSION.exec(readFileSync(repo('build/config.yml'), 'utf8'))?.[1] ?? ''

/** Mortar's own counts, filled into a comparison page's GAME_COUNT and THUNDERSTORE_COUNT. */
export interface Counts {
  games: number
  thunderstore: number
}

/** The Games row's stronger side follows Mortar's live count against the first number in the other manager's cell.
 * An "Over N" figure is not exact: Mortar wins it only by a clear margin, and reads as even short of that. */
const OVER_MARGIN = 1.1

export function gamesWinner(md: string, games: number): string {
  return md.replace(GAMES_ROW, (row, ours: string, theirs: string) => {
    const n = Number(FIRST_NUMBER.exec(theirs)?.[0] ?? Number.NaN)
    const over = OVER_PREFIX.test(theirs)
    if (Number.isNaN(n) || games === n || (over && games >= n && games < n * OVER_MARGIN)) {
      return row
    }
    return games > n ? `| Games | !win ${ours}| ${theirs}|` : `| Games | ${ours}| !win ${theirs}|`
  })
}

/** The line under a manager's name on /why/, from the page's "<!-- tile: ... -->" comment. */
export function whyTile(w: Why): string {
  const what = TILE_COMMENT.exec(w.md)?.[1]
  if (!what) {
    throw new Error(`site: ${WHY}/${w.id}.md needs a "<!-- tile: ... -->" line for /why/`)
  }
  return what
}

/** A comparison's body as HTML: the figures filled in, mod-site tokens as icons and the stronger side of each row marked. */
export function whyHtml(w: Why, counts: Counts): string {
  const filled = w.md
    .replaceAll('GAME_COUNT', String(counts.games))
    .replaceAll('THUNDERSTORE_COUNT', String(counts.thunderstore))
    .replaceAll('MORTAR_VERSION', appVersion())
    .replaceAll(
      'MORTAR_SITES',
      modSites()
        .map((id) => `:${id}:`)
        .join(' '),
    )
    .replace(RIVAL_TOKEN, (token) => {
      const v = rivalFacts()[token.toLowerCase()]
      if (v === undefined) {
        throw new Error(`site: ${w.id}.md names ${token}, which ${RIVALS} lacks`)
      }
      return String(v)
    })
  return (
    sourceIcons(markdown(gamesWinner(filled, counts.games)))
      .replaceAll('<td>!win ', '<td class="win">')
      // An even row highlights both sides: every value cell after the row's label.
      .replace(TABLE_ROW, (row, label: string, cells: string) =>
        cells.includes('class="win"')
          ? row
          : `<tr><td>${label}</td>${cells.replaceAll('<td>', '<td class="win">')}</tr>`,
      )
      .replace(
        '</table></div>',
        '</table></div>\n<p class="table-note">Highlighted: the stronger of the two, or both where they are even.</p>',
      )
  )
}
