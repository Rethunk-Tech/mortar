// Each Steam game's store art (the library capsule for its card, the library hero for its page's banner, the header
// where a game has neither), as the addresses on Steam's CDN its store names. Newer games keep their art under hashed
// paths only the store API reveals, so the addresses are looked up once by `bun scripts/site.ts --refresh` and
// committed, which keeps building the site offline.
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import process from 'node:process'
import { repo } from './site.ts'

const ART = repo('web/src/data/steam-art.json')
const STORE_BROWSE = 'https://api.steampowered.com/IStoreBrowseService/GetItems/v1'
const STORE_ASSETS = 'https://shared.steamstatic.com/store_item_assets/'
const BATCH = 50
const CACHE_BUSTER = /\?t=\d+$/
const FILENAME_TOKEN = /\$\{FILENAME\}/

export interface SteamArt {
  header: string
  hero?: string
  capsule?: string
  capsule2x?: string
}

export function steamArt(): Record<string, SteamArt> {
  return existsSync(ART) ? (JSON.parse(readFileSync(ART, 'utf8')) as Record<string, SteamArt>) : {}
}

interface StoreItem {
  appid: number
  assets?: {
    asset_url_format?: string
    header?: string
    library_hero?: string
    library_capsule?: string
    library_capsule_2x?: string
  }
}

/** Steam stamps every address with a changing `?t=` cache-buster; the committed address stays while only that differs. */
const unstamped = (u: string) => u.replace(CACHE_BUSTER, '')

export async function refreshSteamArt(appIds: string[]): Promise<void> {
  const prev = steamArt()
  const art: Record<string, SteamArt> = {}
  for (let i = 0; i < appIds.length; i += BATCH) {
    const ids = appIds.slice(i, i + BATCH).map((appid) => ({ appid: Number(appid) }))
    const input = JSON.stringify({
      ids,
      context: { country_code: 'US' },
      data_request: { include_assets: true },
    })
    const res = await fetch(`${STORE_BROWSE}?input_json=${encodeURIComponent(input)}`)
    if (!res.ok) {
      throw new Error(`Steam store answered ${res.status}`)
    }
    const body = (await res.json()) as { response?: { store_items?: StoreItem[] } }
    for (const item of body.response?.store_items ?? []) {
      const {
        asset_url_format: format,
        header,
        library_hero: hero,
        library_capsule: capsule,
        library_capsule_2x: capsule2x,
      } = item.assets ?? {}
      if (format && header) {
        const old = Object.values(prev[String(item.appid)] ?? {})
        const url = (file: string) => {
          const fresh = STORE_ASSETS + format.replace(FILENAME_TOKEN, file)
          return old.find((u) => unstamped(u) === unstamped(fresh)) ?? fresh
        }
        art[String(item.appid)] = {
          header: url(header),
          ...(hero ? { hero: url(hero) } : {}),
          ...(capsule ? { capsule: url(capsule) } : {}),
          ...(capsule2x ? { capsule2x: url(capsule2x) } : {}),
        }
      }
    }
  }
  await dropMissing(art)
  const sorted = Object.fromEntries(Object.entries(art).sort(([a], [b]) => Number(a) - Number(b)))
  writeFileSync(ART, `${JSON.stringify(sorted, null, 2)}\n`)
}

const CHECKS_AT_ONCE = 16

/** Whether Steam serves the file: its store API sometimes names art that answers 404. */
async function served(url: string): Promise<boolean> {
  try {
    return (await fetch(url, { method: 'HEAD' })).ok
  } catch {
    return false
  }
}

/** Removes every address Steam does not serve, so a card falls back to the art that loads; a game left without a
 * header has no art at all. */
async function dropMissing(art: Record<string, SteamArt>): Promise<void> {
  const urls = Object.values(art).flatMap((a) => Object.values(a).map(String))
  const missing = new Set<string>()
  for (let i = 0; i < urls.length; i += CHECKS_AT_ONCE) {
    const batch = urls.slice(i, i + CHECKS_AT_ONCE)
    const ok = await Promise.all(batch.map(served))
    batch.forEach((url, j) => {
      if (!ok[j]) {
        missing.add(url)
      }
    })
  }
  for (const [appId, a] of Object.entries(art)) {
    if (missing.has(a.header)) {
      delete art[appId]
    } else {
      art[appId] = Object.fromEntries(
        Object.entries(a).filter(([, url]) => !missing.has(url)),
      ) as SteamArt
    }
  }
  process.stderr.write(
    `site: Steam art: ${urls.length} addresses checked, ${missing.size} not served and dropped\n`,
  )
}
