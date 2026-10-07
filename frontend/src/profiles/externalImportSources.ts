import { useCallback, useEffect, useState } from 'react'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { ExternalSources } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Detection runs each time `active` turns true (a dialog opening), so changes made since the last look show up.
// reload re-runs it, e.g. after the user points Mortar at another Vortex folder.
export function useExternalImportSources(
  game: string,
  active: boolean,
): {
  sources: SourceInfo[]
  reload: () => void
} {
  const [sources, setSources] = useState<SourceInfo[]>([])
  const load = useCallback(
    (isCurrent: () => boolean) =>
      ExternalSources(game)
        .then((detected) => {
          if (isCurrent()) {
            setSources(detected ?? [])
          }
        })
        .catch((error) => {
          if (isCurrent()) {
            setSources([])
            reportUnexpected(error)
          }
        }),
    [game],
  )

  useEffect(() => {
    if (!active) {
      return
    }
    let current = true
    load(() => current)
    return () => {
      current = false
    }
  }, [load, active])

  return { sources, reload: () => load(() => true) }
}
