import { useCallback } from 'react'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { ExternalSources } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useLastGood } from '../shell/useLastGood.ts'
import { reportUnexpected } from '../toasts/report.ts'

const NONE: SourceInfo[] = []

// Detection runs each time `active` turns true (a dialog opening), so changes made since the last look show up; the
// last detection for the game stays on screen meanwhile. reload re-runs it, e.g. after the user points Mortar at
// another Vortex folder.
export function useExternalImportSources(
  game: string,
  active: boolean,
): {
  sources: SourceInfo[]
  reload: () => void
} {
  const { data, reload } = useLastGood<SourceInfo[]>(
    `external-sources:${game}`,
    active ? () => ExternalSources(game).then((detected) => detected ?? []) : null,
    [active],
    { onError: reportUnexpected },
  )
  const rerun = useCallback(() => {
    reload().catch(reportUnexpected)
  }, [reload])
  return { sources: data ?? NONE, reload: rerun }
}
