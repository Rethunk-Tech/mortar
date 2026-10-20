// /faq/ from scripts/site/faq.md, the "Why Mortar?" pages from scripts/site/why/<manager>.md, and the landing page's
// short list of questions that links into the FAQ.
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { brandIcon, shareMarks } from './brands.ts'
import { escapeHTML, page } from './chrome.ts'
import { SOURCE_NAMES } from './games.ts'

const FAQ = 'scripts/site/faq.md'
const WHY = 'scripts/site/why'
const LANDING_QUESTIONS = 4

interface Why {
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
      const name = /^# Why Mortar\? \(vs (.+)\)\n/.exec(md)?.[1]
      if (!name) {
        throw new Error(`site: ${WHY}/${f} must start with "# Why Mortar? (vs <manager>)"`)
      }
      return { id: f.replace(/\.md$/, ''), name, md: md.replace(/^# .*\n/, '') }
    })
}

function render(md: string): string {
  return Bun.markdown
    .html(md, { headings: { ids: true } })
    .replaceAll('<table>', '<div class="table-wrap"><table>')
    .replaceAll('</table>', '</table></div>')
}

/** The FAQ's questions and their Markdown answers, in file order. */
function questions(gameCount: number, whys: Why[]): { q: string; a: string; id: string }[] {
  const whyLinks = whys.map((w) => `- [Why Mortar? (vs ${w.name})](/why/${w.id}/)`).join('\n')
  const md = readFileSync(FAQ, 'utf8')
    .replaceAll('GAME_COUNT', String(gameCount))
    .replaceAll('WHY_LINKS', whyLinks)
  return md
    .split(/^## /m)
    .filter((part) => part.trim() !== '')
    .map((part) => {
      const [q = '', ...rest] = part.split('\n')
      const id =
        Bun.markdown.html(`## ${q}`, { headings: { ids: true } }).match(/id="([^"]+)"/)?.[1] ?? ''
      return { q: q.trim(), a: rest.join('\n').trim(), id }
    })
}

const plain = (md: string) =>
  md
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/\s+/g, ' ')
    .trim()

export function faqPage(gameCount: number, whys: Why[]): string {
  const qs = questions(gameCount, whys)
  const md = qs.map((x) => `## ${x.q}\n\n${x.a}\n`).join('\n')
  return page({
    path: 'site/faq/index.html',
    current: 'faq',
    title: 'Mortar FAQ',
    description:
      'Answers about Mortar: price, games, Nexus Mods, the Steam Deck, downloads and coming from another mod manager.',
    body: `      <h1 class="ext-title">Questions</h1>
      <p class="lede">Short answers about Mortar. The <a href="/guide/">user guide</a> covers everything in detail.</p>
      <div class="faq-page">
${render(md)}
      </div>`,
    jsonLd: {
      '@context': 'https://schema.org',
      '@type': 'FAQPage',
      mainEntity: qs.map((x) => ({
        '@type': 'Question',
        name: x.q,
        acceptedAnswer: { '@type': 'Answer', text: plain(x.a) },
      })),
    },
  })
}

/** A plain Markdown document as a page, titled by its first heading. */
export function docPage(src: string, path: string, description: string): string {
  const md = readFileSync(src, 'utf8')
  const title = /^# (.+)\n/.exec(md)?.[1]
  if (!title) {
    throw new Error(`site: ${src} must start with "# <title>"`)
  }
  return page({
    path,
    title,
    description,
    body: `      <h1 class="ext-title">${escapeHTML(title)}</h1>
${render(md.replace(/^# .*\n/, ''))}`,
  })
}

/** The landing page's FAQ teaser: the first questions, each linking to its answer, and a link to the rest. */
export function landingFaq(gameCount: number, whys: Why[]): string {
  const qs = questions(gameCount, whys)
  const items = qs
    .slice(0, LANDING_QUESTIONS)
    .map((x) => `<li><a href="/faq/#${x.id}">${escapeHTML(x.q)}</a></li>`)
    .join('\n            ')
  return `
          <ul class="faq-links">
            ${items}
          </ul>
          <p class="center"><a class="btn-ghost" href="/faq/">All ${qs.length} questions</a></p>
          `
}

/** A run of `:source:` tokens (`:nexus: :github:`) as one row of mod-site icons. */
const sourceIcons = (html: string) =>
  html.replace(/(?::[a-z]+:\s*)+/g, (run) => {
    const ids = run.match(/[a-z]+/g) ?? []
    if (!ids.every((id) => id in SOURCE_NAMES || id === 'hexium')) {
      return run
    }
    const icons = ids.map((id) => brandIcon(id, SOURCE_NAMES[id] ?? 'Hexium')).join('')
    return `<span class="icon-row">${icons}</span>${run.endsWith(' ') ? ' ' : ''}`
  })

// Facts about other managers read from their own repositories by `bun scripts/site.ts --refresh` (which the release's
// site publish runs too) and committed, so building the site stays offline. A page names a game count as
// R2MODMAN_COUNT, a latest release as MO2_RELEASE and its month as MO2_RELEASED.
const RIVALS = 'scripts/site/rival-facts.json'
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
    count: (body) => captured(body, /support for over (\d+) different games/),
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
  for (const [id, repo] of Object.entries(RIVAL_RELEASES)) {
    const url = `https://api.github.com/repos/${repo}/releases/latest`
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

const rivalFacts = (): Record<string, number | string> =>
  JSON.parse(readFileSync(RIVALS, 'utf8')) as Record<string, number | string>

/** The app version, from build/config.yml where it is set; a comparison page names it as MORTAR_VERSION. */
export const appVersion = (): string =>
  /^\s+version: "([^"]+)"/m.exec(readFileSync('build/config.yml', 'utf8'))?.[1] ?? ''

/** Mortar's own counts, filled into a comparison page's GAME_COUNT and THUNDERSTORE_COUNT. */
interface Counts {
  games: number
  thunderstore: number
}

/** The Games row's stronger side follows Mortar's live count against the first number in the other manager's cell.
 * An "Over N" figure is not exact: Mortar wins it only by a clear margin, and reads as even short of that. */
const OVER_MARGIN = 1.1

export function gamesWinner(md: string, games: number): string {
  return md.replace(/^\| Games \| ([^|]*)\| ([^|]*)\|$/m, (row, ours: string, theirs: string) => {
    const n = Number(/\d+/.exec(theirs)?.[0] ?? Number.NaN)
    const over = /^over /i.test(theirs)
    if (Number.isNaN(n) || games === n || (over && games >= n && games < n * OVER_MARGIN)) {
      return row
    }
    return games > n ? `| Games | !win ${ours}| ${theirs}|` : `| Games | ${ours}| !win ${theirs}|`
  })
}

/** /why/: a tile per manager Mortar is compared with, its app icon over its name. */
export function whyIndexPage(whys: Why[]): string {
  const curseforge = shareMarks().curseforge
  const tiles = whys.map((w) => {
    const icon =
      w.id === 'curseforge' && curseforge && 'colour' in curseforge
        ? `<svg class="rival-icon" style="color: ${curseforge.colour}" aria-hidden="true"><use href="/img/brands.svg#curseforge" /></svg>`
        : `<img class="rival-icon" src="/img/managers/${w.id}.png" alt="" width="56" height="56" />`
    return `          <li><a class="rival" href="/why/${w.id}/">${icon}<span>${escapeHTML(w.name.replace(/^the /, ''))}</span></a></li>`
  })
  return page({
    path: 'site/why/index.html',
    title: 'Why Mortar? Compared with other mod managers',
    description:
      'How Mortar compares with other mod managers, with sources for every claim about them.',
    body: `      <h1 class="ext-title">Why Mortar?</h1>
      <p class="lede">Pick the manager you use now to see how Mortar compares, with a source for every claim about it.</p>
      <ul class="rival-grid">
${tiles.join('\n')}
      </ul>`,
  })
}

export function whyPage(w: Why, counts: Counts): string {
  const filled = w.md
    .replaceAll('GAME_COUNT', String(counts.games))
    .replaceAll('THUNDERSTORE_COUNT', String(counts.thunderstore))
    .replaceAll('MORTAR_VERSION', appVersion())
    .replace(/\b[A-Z0-9]+_(?:COUNT|RELEASED?)\b/g, (token) => {
      const v = rivalFacts()[token.toLowerCase()]
      if (v === undefined) {
        throw new Error(`site: ${w.id}.md names ${token}, which ${RIVALS} lacks`)
      }
      return String(v)
    })
  const md = gamesWinner(filled, counts.games)
  return page({
    path: `site/why/${w.id}/index.html`,
    title: `Why Mortar? (vs ${w.name})`,
    description: `How Mortar compares with ${w.name}, with sources for every claim about ${w.name}.`,
    body: `      <h1 class="ext-title">Why Mortar? (vs ${escapeHTML(w.name)})</h1>
${sourceIcons(render(md))
  .replaceAll('<td>!win ', '<td class="win">')
  // An even row highlights both sides: every value cell after the row's label.
  .replace(/<tr><td>(.*?)<\/td>(.*?)<\/tr>/g, (row, label: string, cells: string) =>
    cells.includes('class="win"')
      ? row
      : `<tr><td>${label}</td>${cells.replaceAll('<td>', '<td class="win">')}</tr>`,
  )
  .replace(
    '</table></div>',
    '</table></div>\n<p class="table-note">Highlighted: the stronger of the two, or both where they are even.</p>',
  )}
      <div class="cta ext-cta">
        <a class="btn" href="/download/">Download Mortar</a>
        <a class="btn-ghost" href="/faq/">All questions</a>
      </div>`,
  })
}
