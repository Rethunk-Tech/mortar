import { useEffect, useState } from 'react'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/migrate/models.ts'
import { ExternalSources } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function useExternalImportSources(game: string): SourceInfo[] {
  const [sources, setSources] = useState<SourceInfo[]>([])

  useEffect(() => {
    let current = true
    ExternalSources(game)
      .then((detected) => {
        if (current) {
          setSources(detected ?? [])
        }
      })
      .catch((error) => {
        if (current) {
          setSources([])
          reportUnexpected(error)
        }
      })
    return () => {
      current = false
    }
  }, [game])

  return sources
}
