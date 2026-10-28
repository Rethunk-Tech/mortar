import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetModFavourite } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

/** How a starred mod is named: its source and the id the source gives it, in lower case (settings.FavouriteKey). */
function favouriteKey(source: string, id: string): string {
  return `${source}:${id}`.toLowerCase()
}

/** The key of the mod an installed entry came from, matching its Browse hit; empty for a mod with no site. */
function entryFavouriteKey(entry: Entry | undefined): string {
  const src = entry?.source
  if (!(entry && src)) {
    return ''
  }
  if (src.kind === 'nexus') {
    return src.modId ? favouriteKey('nexus', String(src.modId)) : ''
  }
  if (src.kind === 'local') {
    return src.name ? favouriteKey('local', `${src.name}|${entry.item || entry.key}`) : ''
  }
  const id = src.kind === 'github' ? src.repo : src.name
  return id ? favouriteKey(src.kind, id) : ''
}

/** Whether the game stars the mod; the list is the game's favouriteMods setting. */
function isStarred(starred: readonly string[] | null | undefined, key: string): boolean {
  const list = starred ?? []
  // A local archive's key leads with the file name that 0.3.8 starred it by (settings.LocalFavouriteKey).
  return (
    key !== '' &&
    (list.includes(key) || (key.startsWith('local:') && list.includes(key.split('|')[0] ?? '')))
  )
}

function useStarred(game: string, key: string): boolean {
  return useSettings((s) => isStarred(s.games?.[game]?.favouriteMods, key))
}

function setStarred(game: string, key: string, on: boolean) {
  SetModFavourite(game, key, on).catch(reportUnexpected)
}

export { entryFavouriteKey, favouriteKey, isStarred, setStarred, useStarred }
