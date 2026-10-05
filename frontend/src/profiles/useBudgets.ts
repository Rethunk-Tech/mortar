import { useEffect, useState } from 'react'
import type { ProfileBudget } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import { ProfileBudgets } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

// Disk use per profile of the open game. The backend caches the measure, so reading again after the profiles change
// (the stamp) costs a lookup, not a walk.
export function useBudgets(game: string, stamp: string): Map<string, ProfileBudget> {
  const [byId, setById] = useState<Map<string, ProfileBudget>>(new Map())
  useEffect(() => {
    if (game === '' || stamp === '') {
      return
    }
    let live = true
    ProfileBudgets().then((all) => {
      if (live) {
        setById(new Map((all ?? []).filter((b) => b.game === game).map((b) => [b.id, b])))
      }
    }, reportUnexpected)
    return () => {
      live = false
    }
  }, [game, stamp])
  return byId
}
