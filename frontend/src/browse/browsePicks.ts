import type { GameSettings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { persist } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import type { BrowseFilter } from './browseTypes.ts'

// What the player picked in Browse for a game: the categories to include and exclude, and the sort.
type BrowsePicks = Pick<BrowseFilter, 'include' | 'exclude' | 'sort'>

const strings = (v: unknown): string[] =>
  Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []

// The saved picks, or null for a game whose picker was never edited, which is when the default excludes apply.
export function parsePicks(saved: string): BrowsePicks | null {
  if (saved === '') {
    return null
  }
  try {
    const raw: unknown = JSON.parse(saved)
    if (typeof raw !== 'object' || raw === null) {
      return null
    }
    const o = raw as Record<string, unknown>
    return {
      include: strings(o.include),
      exclude: strings(o.exclude),
      sort: typeof o.sort === 'string' ? o.sort : '',
    }
  } catch {
    return null
  }
}

export function formatPicks(p: BrowsePicks): string {
  return JSON.stringify({ include: p.include, exclude: p.exclude, sort: p.sort })
}

export function savePicks(game: string, picks: BrowsePicks, failure: string) {
  const value = formatPicks(picks)
  useSettings.setState((s) => ({
    games: { ...s.games, [game]: { ...s.games?.[game], browsePicks: value } as GameSettings },
  }))
  persist(() => SetByKey('browsePicks', value, game), failure)
}
