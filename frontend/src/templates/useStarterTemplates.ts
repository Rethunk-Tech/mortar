import type { StarterTemplate } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import { StarterTemplates } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'

/** The catalog's built-in starter templates for the game, read when `active` turns on. */
// One empty list for every reset: a fresh [] would redraw the closed dialog each time the game it is for changes.
const NONE: StarterTemplate[] = []

export function useStarterTemplates(game: string, active: boolean): StarterTemplate[] {
  const { data: starters } = useLoaded<StarterTemplate[]>(
    active && game ? () => StarterTemplates(game).then((found) => found ?? []) : null,
    [game, active],
    NONE,
    reportUnexpected,
  )
  return starters
}
