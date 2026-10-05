import { useEffect, useState } from 'react'
import type { StarterTemplate } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import { StarterTemplates } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/templates/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

/** The catalog's built-in starter templates for the game, read when `active` turns on. */
export function useStarterTemplates(game: string, active: boolean): StarterTemplate[] {
  const [starters, setStarters] = useState<StarterTemplate[]>([])
  useEffect(() => {
    if (!(active && game)) {
      return
    }
    let live = true
    StarterTemplates(game).then((found) => live && setStarters(found ?? []), reportUnexpected)
    return () => {
      live = false
    }
  }, [game, active])
  return starters
}
