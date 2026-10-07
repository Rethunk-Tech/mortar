import type { ProfileBudget } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import { ProfileBudgets } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Disk use per profile of the open game. The backend caches the measure, so reading again after the profiles change
// (the stamp) costs a lookup, not a walk.
export function useBudgets(game: string, stamp: string): Map<string, ProfileBudget> {
  const { data: byId } = useLoaded<Map<string, ProfileBudget>>(
    game === '' || stamp === ''
      ? null
      : () =>
          ProfileBudgets().then(
            (all) => new Map((all ?? []).filter((b) => b.game === game).map((b) => [b.id, b])),
          ),
    [game, stamp],
    new Map(),
    reportUnexpected,
  )
  return byId
}
