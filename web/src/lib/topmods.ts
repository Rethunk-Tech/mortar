// The most downloaded mods of each game on the mod sites that answer without a key, and a game's optional hand-written
// note (web/src/content/games/<game>.md). The mods are looked up by `bun run site:refresh` and committed, which keeps
// building the site offline; a game's page quotes nothing that is not in the data file.
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { type Game, SOURCE_NAMES } from './games.ts'
import { markdown, repo } from './site.ts'

const TOP = repo('web/src/data/top-mods.json')
const SHOWN = 5
const AT_ONCE = 8
const SUMMARY_CHARS = 120
const THUNDERSTORE = 'https://thunderstore.io/api/cyberstorm/listing'
const VSMODDB = 'https://mods.vintagestory.at/api/mods?orderby=downloads'
// Sources whose lists need an account key, which a site build does not hold.
const KEYED = new Set(['nexus', 'curseforge', 'modio'])

export interface TopMod {
  name: string
  author: string
  summary: string
  downloads: number
  url: string
  icon?: string
}

/** A game's top mods by source id. */
export type GameTop = Record<string, TopMod[]>

let cached: Record<string, GameTop> | undefined

export function topMods(): Record<string, GameTop> {
  if (!cached) {
    cached = existsSync(TOP)
      ? (JSON.parse(readFileSync(TOP, 'utf8')) as Record<string, GameTop>)
      : {}
  }
  return cached
}

/** The game's note as HTML, or undefined when nobody wrote one. */
export function gameNote(id: string): string | undefined {
  const file = `${repo('web/src/content/games')}/${id}.md`
  return existsSync(file) ? markdown(readFileSync(file, 'utf8')) : undefined
}

interface TsPackage {
  namespace: string
  name: string
  description: string
  download_count: number
  icon_url?: string
  is_pinned: boolean
  is_deprecated: boolean
  is_nsfw: boolean
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(`${url} answered ${res.status}`)
  }
  return (await res.json()) as T
}

/** A summary cut at a word to fit a list row. */
function brief(text: string): string {
  const one = text.replace(/\s+/g, ' ').trim()
  return one.length <= SUMMARY_CHARS ? one : `${one.slice(0, one.lastIndexOf(' ', SUMMARY_CHARS))}…`
}

async function thunderstore(community: string): Promise<TopMod[]> {
  const body = await getJSON<{ results: TsPackage[] }>(
    `${THUNDERSTORE}/${encodeURIComponent(community)}/?ordering=most-downloaded&page=1`,
  )
  // A pinned package is the community's mod manager or loader pack, which every player already has.
  return body.results
    .filter((p) => !(p.is_pinned || p.is_deprecated || p.is_nsfw))
    .slice(0, SHOWN)
    .map((p) => ({
      name: p.name.replaceAll('_', ' '),
      author: p.namespace,
      summary: brief(p.description),
      downloads: p.download_count,
      url: `https://thunderstore.io/c/${community}/p/${p.namespace}/${p.name}/`,
      ...(p.icon_url ? { icon: p.icon_url } : {}),
    }))
}

async function vsmoddb(): Promise<TopMod[]> {
  const body = await getJSON<{
    mods: {
      modid: number
      name: string
      author: string
      summary: string
      downloads: number
      logo?: string
    }[]
  }>(VSMODDB)
  return body.mods.slice(0, SHOWN).map((m) => ({
    name: m.name,
    author: m.author,
    summary: brief(m.summary),
    downloads: m.downloads,
    url: `https://mods.vintagestory.at/show/mod/${m.modid}`,
    ...(m.logo ? { icon: m.logo } : {}),
  }))
}

/** A source that cannot be read keeps its committed list, so one site being down never blocks a release's publish. */
export async function refreshTopMods(games: Game[]): Promise<void> {
  const prev = topMods()
  const out: Record<string, GameTop> = {}
  const jobs: { game: string; source: string; run: () => Promise<TopMod[]> }[] = []
  const skipped = new Set<string>()
  for (const g of games) {
    for (const s of g.sources ?? []) {
      if (s.id === 'thunderstore' && s.key) {
        const community = s.key
        jobs.push({ game: g.id, source: s.id, run: () => thunderstore(community) })
      } else if (s.id === 'vsmoddb') {
        jobs.push({ game: g.id, source: s.id, run: vsmoddb })
      } else if (KEYED.has(s.id)) {
        skipped.add(s.id)
      }
    }
  }
  let kept = 0
  for (let i = 0; i < jobs.length; i += AT_ONCE) {
    await Promise.all(
      jobs.slice(i, i + AT_ONCE).map(async (job) => {
        let mods: TopMod[] | undefined
        try {
          mods = await job.run()
        } catch (err) {
          mods = prev[job.game]?.[job.source]
          kept += 1
          process.stderr.write(
            `site: top mods of ${job.game} on ${job.source}: ${mods ? 'kept the committed list: ' : 'none: '}${String(err)}\n`,
          )
        }
        if (mods && mods.length > 0) {
          out[job.game] = { ...out[job.game], [job.source]: mods }
        }
      }),
    )
  }
  const sorted = Object.fromEntries(Object.entries(out).sort(([a], [b]) => a.localeCompare(b)))
  writeFileSync(TOP, `${JSON.stringify(sorted, null, 2)}\n`)
  cached = sorted
  process.stderr.write(
    `site: top mods: ${Object.keys(sorted).length} games from ${jobs.length} lists, ${kept} kept from the committed file; ${[...skipped].map((s) => SOURCE_NAMES[s] ?? s).join(', ')} skipped, since their lists need an account key\n`,
  )
}
