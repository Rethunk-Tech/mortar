import { useLingui } from '@lingui/react/macro'
import { useLoaded } from '@rethunk/react-load-hooks'
import type { EverywherePreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PreviewEverywhere } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { updateSources, useOfflineReason } from '../shell/offlineText.ts'

// What "Update in all profiles" would touch, and why it cannot run when it cannot (empty when it can).
export function useEverywhere(
  game: string,
  id: string,
  update: { source: string; nexusId: number },
) {
  const { t } = useLingui()
  // A preview that cannot be read leaves the action without its count.
  const { data: preview } = useLoaded<EverywherePreview | null>(
    game === '' || id === '' ? null : () => PreviewEverywhere(game, id),
    [game, id],
    null,
  )
  const count = preview?.affected?.length ?? 0
  const offline = useOfflineReason(updateSources(update))
  let why = ''
  if (offline !== '') {
    why = offline
  } else if (preview === null) {
    why = t`Checking profiles…`
  } else if (count === 0) {
    why = t`No eligible profiles.`
  }
  return { count, why }
}
