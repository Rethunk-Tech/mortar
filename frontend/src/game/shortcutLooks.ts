import { create } from 'zustand'
import { Exists } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shortcut/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Whether each profile has a desktop shortcut, by game and profile. A look that finds what was known sets nothing, so
// the menu items showing it do not redraw.
const useShortcuts = create<Record<string, boolean>>(() => ({}))
const key = (game: string, profile: string) => `${game}/${profile}`
// A look this recent stands for the next: a menu's items look as they mount and the menu again as it opens.
const FRESH_MS = 1000
const lookedAt = new Map<string, number>()

export const setShortcut = (game: string, profile: string, has: boolean) => {
  if (useShortcuts.getState()[key(game, profile)] !== has) {
    useShortcuts.setState({ [key(game, profile)]: has })
  }
}

/** Looks for the profile's shortcut again: the file may have gone, or been made, since the last look. */
export const lookForShortcut = (game: string, profile: string) => {
  const k = key(game, profile)
  if (Date.now() - (lookedAt.get(k) ?? 0) < FRESH_MS) {
    return
  }
  lookedAt.set(k, Date.now())
  Exists(game, profile)
    .then((has) => setShortcut(game, profile, has))
    .catch(reportUnexpected)
}

export const useHasShortcut = (game: string, profile: string) =>
  useShortcuts((s) => s[key(game, profile)] ?? false)
