import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { SetDevProjectOrder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { placeInSlots } from './projectOrder.ts'

function setProjects(game: string, devProjects: DevProject[]) {
  useSettings.setState((s) => {
    const prefs = s.games?.[game]
    return prefs ? { games: { ...s.games, [game]: { ...prefs, devProjects } } } : s
  })
}

/** Shows the new order at once and saves it; a refused save puts the old order back. */
export function saveOrder(game: string, dirs: string[]) {
  const before = useSettings.getState().games?.[game]?.devProjects
  if (!before) {
    return
  }
  setProjects(game, placeInSlots(before, dirs))
  SetDevProjectOrder(game, dirs).catch((e: unknown) => {
    setProjects(game, before)
    reportUnexpected(e)
  })
}
