import { create } from 'zustand'
import { Kinds } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Settings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

/** The asset kind of each project made only of files, by folder; a code project has no entry. The files decide it, not the settings. */
export const useDevKinds = create<{ byDir: Record<string, string> }>(() => ({ byDir: {} }))

/** Whether the project in dir is made only of files. */
export const useIsAsset = (dir: string) => useDevKinds((s) => Boolean(s.byDir[dir]))

async function refreshKinds(game: string) {
  const found: Record<string, string> = {}
  for (const [dir, kind] of Object.entries((await Kinds(game)) ?? {})) {
    if (kind) {
      found[dir] = kind
    }
  }
  useDevKinds.setState((s) => ({ byDir: { ...s.byDir, ...found } }))
}

type Games = Settings['games']

/** Reads the kinds at start and again whenever a game's project list changes. */
export function initDevKinds() {
  const reread = (games: Games, before: Games) => {
    for (const game of Object.keys(games ?? {})) {
      if (games?.[game]?.devProjects !== before?.[game]?.devProjects) {
        refreshKinds(game).catch(reportUnexpected)
      }
    }
  }
  useSettings.subscribe((s, prev) => reread(s.games, prev.games))
  reread(useSettings.getState().games, {})
}
