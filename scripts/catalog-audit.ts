// Compares the catalog with the mod sites, run by hand before a release (`wails3 task components:audit`): lists the
// listed Thunderstore communities with enough packages that the catalog lacks, lists catalog games that Nexus or
// CurseForge host but the catalog does not name them for, and fails when a catalog game's community or loader pack
// is gone.
import { readFileSync } from 'node:fs'

const SOURCE = 'components.source.json'
const API = 'https://thunderstore.io/api'
// A community smaller than this is not worth a catalog entry yet.
const MIN_PACKAGES = 20

interface Game {
  id: string
  name: string
  sources?: { id: string; key?: string }[]
  loaders?: { id: string; package?: string }[]
}
interface Community {
  identifier: string
  name: string
  is_listed: boolean
  total_package_count: number
}

async function json<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { 'User-Agent': 'Mortar catalog audit' } })
  if (!res.ok) {
    throw new Error(`${url}: ${res.status}`)
  }
  return (await res.json()) as T
}

async function packageExists(id: string): Promise<boolean> {
  const cut = id.indexOf('-')
  const res = await fetch(`${API}/experimental/package/${id.slice(0, cut)}/${id.slice(cut + 1)}/`, {
    method: 'HEAD',
  })
  return res.ok
}

const games: Game[] = JSON.parse(readFileSync(SOURCE, 'utf8')).games
const keyed = new Map<string, string>()
for (const g of games) {
  const key = g.sources?.find((s) => s.id === 'thunderstore')?.key
  if (key) {
    keyed.set(key, g.id)
  }
}
const { results } = await json<{ results: Community[] }>(`${API}/cyberstorm/community/`)
const live = new Set(results.map((c) => c.identifier))

const missing = results
  .filter((c) => c.is_listed && !keyed.has(c.identifier) && c.total_package_count >= MIN_PACKAGES)
  .toSorted((a, b) => b.total_package_count - a.total_package_count)
console.log(
  `${missing.length} listed communities with ${MIN_PACKAGES}+ packages are not in the catalog:`,
)
for (const c of missing) {
  console.log(`  ${c.identifier} (${c.name}): ${c.total_package_count} packages`)
}

const drift: string[] = []
for (const [key, id] of keyed) {
  if (!live.has(key)) {
    drift.push(`${id}: Thunderstore has no community "${key}"`)
  }
}
const packs = new Set(
  games.flatMap((g) => (g.loaders ?? []).map((l) => l.package ?? '')).filter(Boolean),
)
for (const pack of packs) {
  if (!(await packageExists(pack))) {
    drift.push(`loader pack ${pack} is not on Thunderstore`)
  }
}
console.log(
  drift.length === 0 ? 'Every catalog community and loader pack is live.' : drift.join('\n'),
)
process.exitCode = drift.length === 0 ? 0 : 1

// A Nexus source is added only when the site's mods are for the game's own loader: the loader named at least as often
// as any other in the mods' descriptions. These games fail that, or name no loader, so they stay out until it changes.
const NEXUS_LEFT_OUT = new Set([
  'sons-of-the-forest',
  'ai-limit',
  'blue-prince',
  'gunman-contracts-stand-alone',
  'the-spell-brigade',
  'everything-is-crab',
  'grey-hack',
  'wizard-with-a-gun',
  'against',
  'gorilla-tag',
  'dale-dawson-stationery-supplies',
  'pit-of-goblin',
  'scrap-mechanic',
  'chrono-ark',
  'lethal-league-blaze',
  'wrestling-empire',
  'gatekeeper',
  'in-falsus',
  'pac-man-world-re-pac',
  'atomicrops',
  'below-the-stone',
  'crawlspace-2',
  'five-nights-at-freddys-into-the-pit',
  'green-hell-vr',
  'guilty-as-sock',
  'holdfast-nations-at-war',
  'human-fall-flat',
  'i-am-your-beast',
  'last-train-outta-wormtown',
  'necropolis',
  'pulsar-lost-colony',
  'jump-scare-mansion',
  'west-of-loathing',
  'word-play',
  'among-us',
  'core-keeper',
  'patapon-1-2-replay',
  'wildfrost',
  'hard-bullet',
  'patch-quest',
  'beton-brutal',
])
const LOADER_WORDS: Record<string, string> = {
  bepinex5: 'BepInEx',
  melonloader: 'MelonLoader',
  lovely: 'Steamodded',
  returnofmodding: 'Hell2Modding',
  northstar: 'Northstar',
  gdweave: 'GDWeave',
  rivet: 'Rivet',
  umm: 'Unity Mod Manager',
  shimloader: 'UE4SS',
}
const RIVAL_WORDS = ['BepInEx', 'MelonLoader', 'RedLoader', 'Unity Mod Manager', 'UE4SS']
const norm = (s: string) =>
  s
    .toLowerCase()
    .replace(/^the /, '')
    .replaceAll('&', 'and')
    .replace(/[^a-z0-9]/g, '')
const has = (g: Game, site: string) => g.sources?.some((s) => s.id === site) ?? false

async function nexus<T>(query: string): Promise<T> {
  const res = await fetch('https://api.nexusmods.com/v2/graphql', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'User-Agent': 'Mortar catalog audit' },
    body: JSON.stringify({ query }),
  })
  if (!res.ok) {
    throw new Error(`Nexus: ${res.status}`)
  }
  return ((await res.json()) as { data: T }).data
}

interface NexusGame {
  id: number
  domainName: string
  name: string
  modCount: number
}
const nexusGames = new Map<string, NexusGame>()
for (let offset = 0; ; offset += 1000) {
  const { games: page } = await nexus<{ games: { totalCount: number; nodes: NexusGame[] } }>(
    `{ games(count: 1000, offset: ${offset}) { totalCount nodes { id domainName name modCount } } }`,
  )
  for (const g of page.nodes) {
    const k = norm(g.name)
    if ((nexusGames.get(k)?.modCount ?? -1) < g.modCount) {
      nexusGames.set(k, g)
    }
  }
  if (page.nodes.length === 0 || offset + page.nodes.length >= page.totalCount) {
    break
  }
}
const mentions = async (domain: string, word: string) =>
  (
    await nexus<{ mods: { totalCount: number } }>(
      `{ mods(filter: { gameDomainName: [{ value: "${domain}", op: EQUALS }], description: [{ value: "${word}", op: MATCHES }] }) { totalCount } }`,
    )
  ).mods.totalCount
const toAdd: string[] = []
for (const g of games) {
  const n = nexusGames.get(norm(g.name))
  const word = LOADER_WORDS[g.loaders?.[0]?.id ?? '']
  if (has(g, 'nexus') || NEXUS_LEFT_OUT.has(g.id) || !n || n.modCount === 0 || !word) {
    continue
  }
  const own = await mentions(n.domainName, word)
  let best = own
  for (const rival of RIVAL_WORDS.filter((w) => w !== word)) {
    best = Math.max(best, await mentions(n.domainName, rival))
  }
  if (own > 0 && own >= best) {
    toAdd.push(
      `  ${g.id}: { "id": "nexus", "key": "${n.domainName}", "gameId": ${n.id} } (${n.modCount} mods)`,
    )
  }
}
console.log(`${toAdd.length} catalog games Nexus hosts for their loader have no nexus source:`)
for (const line of toAdd) {
  console.log(line)
}

const key = process.env.MORTAR_CURSEFORGE_KEY
if (key) {
  const cf = async <T>(path: string) => {
    const res = await fetch(`https://api.curseforge.com/v1${path}`, {
      headers: { 'x-api-key': key },
    })
    if (!res.ok) {
      throw new Error(`CurseForge ${path}: ${res.status}`)
    }
    return ((await res.json()) as { data: T }).data
  }
  const cfGames = new Map(
    (await cf<{ id: number; name: string }[]>('/games?pageSize=50')).map((g) => [norm(g.name), g]),
  )
  const cfMissing = games.filter((g) => !has(g, 'curseforge') && cfGames.has(norm(g.name)))
  console.log(`${cfMissing.length} catalog games CurseForge hosts have no curseforge source:`)
  for (const g of cfMissing) {
    const c = cfGames.get(norm(g.name))
    const classes = await cf<{ id: number; name: string }[]>(
      `/categories?gameId=${c?.id}&classesOnly=true`,
    )
    console.log(
      `  ${g.id}: gameId ${c?.id}, classes ${classes.map((k) => `${k.id} ${k.name}`).join(', ')}`,
    )
  }
} else {
  console.log('MORTAR_CURSEFORGE_KEY is unset: CurseForge not checked.')
}
