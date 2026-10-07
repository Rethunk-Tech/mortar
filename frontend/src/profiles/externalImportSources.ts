import { useCallback, useEffect, useState } from 'react'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { ExternalSources } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// reload re-runs detection, e.g. after the user points Mortar at another Vortex folder.
export function useExternalImportSources(game: string): {
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
    let current = true
    load(() => current)
    return () => {
      current = false
    }
  }, [load])

  return { sources, reload: () => load(() => true) }
}
