// Compares the catalog with the mod sites, run by hand before a release (`wails3 task components:audit`): lists the
// listed Thunderstore communities with enough packages that the catalog lacks, lists catalog games that Nexus or
// CurseForge host but the catalog does not name them for, and fails when a catalog game's community or loader pack
// is gone.

const SOURCE = 'components.source.json'
const API = 'https://thunderstore.io/api'
// A community smaller than this is not worth a catalog entry yet.
const MIN_PACKAGES = 20
const PAGE_SIZE = 1000
const NOT_FOUND = 404
const TOO_MANY_REQUESTS = 429
const SERVER_ERROR = 500
const RETRY_DELAY_MS = 10_000
const ATTEMPTS = 18

// The sites answer a 429 or a 5xx now and then, sometimes for minutes, and a failed answer must not read as a missing
// package: each request is tried every 10 seconds for three minutes before its answer counts.
async function fetchRetry(url: string, init?: RequestInit): Promise<Response> {
  for (let attempt = 1; ; attempt++) {
    try {
      const res = await fetch(url, init)
      if (attempt === ATTEMPTS || (res.status !== TOO_MANY_REQUESTS && res.status < SERVER_ERROR)) {
        return res
      }
    } catch (err) {
      if (attempt === ATTEMPTS) {
        throw err
      }
    }
    await Bun.sleep(RETRY_DELAY_MS)
  }
}

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
  const res = await fetchRetry(url, { headers: { 'User-Agent': 'Mortar catalog audit' } })
  if (!res.ok) {
    throw new Error(`${url}: ${res.status}`)
  }
  return (await res.json()) as T
}

async function packageExists(id: string): Promise<boolean> {
  const cut = id.indexOf('-')
  const url = `${API}/experimental/package/${id.slice(0, cut)}/${id.slice(cut + 1)}/`
  const res = await fetchRetry(url, { method: 'HEAD' })
  if (res.status === NOT_FOUND) {
    return false
  }
  if (!res.ok) {
    throw new Error(`${url}: ${res.status}`)
  }
  return true
}

const games: Game[] = (await Bun.file(SOURCE).json()).games
const keyed = new Map<string, string>()
for (const g of games) {
  const key = g.sources?.find((s) => s.id === 'thunderstore')?.key
  if (key) {
    keyed.set(key, g.id)
  }
}
const { results } = await json<{ results: Community[] }>(`${API}/cyberstorm/community/`)
const live = new Set(results.map((c) => c.identifier))

// The list leaves out unlisted communities, which still serve their packages, so only the community's own page can say
// it is gone.
async function communityExists(community: string): Promise<boolean> {
  const res = await fetchRetry(`https://thunderstore.io/c/${community}/`, { method: 'HEAD' })
  return res.status !== NOT_FOUND
}

// Communities that stay out of the catalog, each for the reason beside it.
const LEFT_OUT = new Set([
  // Its Thunderstore page says mods moved to the Steam Workshop and those left there are out of date.
  'wrestling-empire',
  // Its Thunderstore page says mods moved to the Steam Workshop and those left there are out of date.
  'stacklands',
  // Neither r2modman nor Gale offers these: the schema gives them no r2modman section, so there is no loader or install
  // rule to follow.
  'reskate',
  'zelda-64-recompiled',
  'banjo-recompiled',
  '2-ship-2-harkinian',
  'r5flowstate',
  // Neither r2modman nor Gale lists it, and no store Mortar reads sells it.
  '3dash',
])

const missing = results
  .filter(
    (c) =>
      c.is_listed &&
      !keyed.has(c.identifier) &&
      !LEFT_OUT.has(c.identifier) &&
      c.total_package_count >= MIN_PACKAGES,
  )
  .toSorted((a, b) => b.total_package_count - a.total_package_count)
console.log(
  `${missing.length} listed communities with ${MIN_PACKAGES}+ packages are not in the catalog:`,
)
for (const c of missing) {
  console.log(`  ${c.identifier} (${c.name}): ${c.total_package_count} packages`)
}

const drift: string[] = []
for (const [key, id] of keyed) {
  if (!(live.has(key) || (await communityExists(key)))) {
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
  'patapon-1-2-replay',
  'wildfrost',
  'hard-bullet',
  'patch-quest',
  'beton-brutal',
])
const LOADER_WORDS: Record<string, string> = {
  bepinex5: 'BepInEx',
  bepisloader: 'BepisLoader',
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
  const res = await fetchRetry('https://api.nexusmods.com/v2/graphql', {
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
for (let offset = 0; ; offset += PAGE_SIZE) {
  const { games: page } = await nexus<{ games: { totalCount: number; nodes: NexusGame[] } }>(
    `{ games(count: ${PAGE_SIZE}, offset: ${offset}) { totalCount nodes { id domainName name modCount } } }`,
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
    const res = await fetchRetry(`https://api.curseforge.com/v1${path}`, {
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
