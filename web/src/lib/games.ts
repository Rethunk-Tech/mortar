// The playable games in the bundled catalog and the facts the site shows about each.
import { readFileSync } from 'node:fs'
import type { SteamArt } from './art.ts'
import { listed, repo } from './site.ts'

const CATALOG = repo('internal/components/components.json')

export interface Game {
  id: string
  name: string
  enabled: boolean
  // A dedicated server's profile, which shares its client's page rather than being a game of its own.
  server?: unknown
  stores?: Record<string, unknown>
  loaders?: { id: string; name: string }[]
  sources?: { id: string; key?: string }[]
}

export const STORE_NAMES: Record<string, string> = {
  steam: 'Steam',
  gog: 'GOG',
  epic: 'Epic Games',
  ea: 'EA app',
  xbox: 'Xbox app',
  lutris: 'Lutris',
  bottles: 'Bottles',
  flatpak: 'Flatpak',
  standalone: 'Its own installer',
}

export const SOURCE_NAMES: Record<string, string> = {
  nexus: 'Nexus Mods',
  thunderstore: 'Thunderstore',
  curseforge: 'CurseForge',
  modrinth: 'Modrinth',
  github: 'GitHub',
  vsmoddb: 'Vintage Story Mod DB',
  itch: 'itch.io',
  patreon: 'Patreon',
}

let playable: Game[] | undefined

export function playableGames(): Game[] {
  if (!playable) {
    const { games } = JSON.parse(readFileSync(CATALOG, 'utf8')) as { games: Game[] }
    playable = games.filter((g) => g.enabled && g.server === undefined)
  }
  return playable
}

/** The mod sites any playable game downloads from, in first-seen catalog order. */
export function modSites(): string[] {
  return [...new Set(playableGames().flatMap((g) => (g.sources ?? []).map((s) => s.id)))]
}

export function named(ids: string[], names: Record<string, string>): string[] {
  return ids.map((id) => {
    const name = names[id]
    if (!name) {
      throw new Error(`site: no display name for "${id}"; add it to web/src/lib/games.ts`)
    }
    return name
  })
}

export function facts(g: Game) {
  const loaders = (g.loaders ?? []).map((l) => l.name).filter((n) => n !== '')
  return {
    loader: listed(loaders),
    stores: listed(named(Object.keys(g.stores ?? {}), STORE_NAMES)),
    sources: listed(
      named(
        (g.sources ?? []).map((s) => s.id),
        SOURCE_NAMES,
      ),
    ),
  }
}

export const steamAppId = (g: Game) => (g.stores?.steam as { appId?: string } | undefined)?.appId

/** The game's own page on a mod site, where the catalog key names one. */
export function sourceURL(s: { id: string; key?: string }): string | undefined {
  switch (s.id) {
    case 'thunderstore':
      return s.key ? `https://thunderstore.io/c/${s.key}/` : undefined
    case 'nexus':
      return s.key ? `https://www.nexusmods.com/games/${s.key}/mods` : undefined
    case 'vsmoddb':
      return 'https://mods.vintagestory.at/'
    default:
      return undefined
  }
}

const MORE_GAMES = 4

/** The games after g in name order that share its loader or, for a loader only g runs, its first mod site. */
export function neighbours(g: Game, games: Game[]): { title: string; games: Game[] } {
  const { loader } = facts(g)
  let same = games.filter((o) => facts(o).loader === loader)
  let title = loader ? `More games on ${loader}` : 'More games without a mod loader'
  const site = g.sources?.[0]?.id
  if (same.length < 2 && site) {
    same = games.filter((o) => o.sources?.some((s) => s.id === site))
    title = `More games with mods from ${SOURCE_NAMES[site] ?? site}`
  }
  same.sort((a, b) => a.name.localeCompare(b.name))
  const at = same.findIndex((o) => o.id === g.id)
  const count = Math.min(MORE_GAMES, same.length - 1)
  return {
    title,
    games: Array.from({ length: count }, (_, i) => same[(at + 1 + i) % same.length]).filter(
      (o): o is Game => o !== undefined,
    ),
  }
}

/** The game's Steam art, where the store lists it. */
export function artOf(g: Game, art: Record<string, SteamArt>): SteamArt | undefined {
  const appId = steamAppId(g)
  return appId ? art[appId] : undefined
}
