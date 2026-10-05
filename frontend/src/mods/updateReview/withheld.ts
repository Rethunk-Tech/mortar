import { msg } from '@lingui/core/macro'
import { useEffect } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { i18n } from '../../i18n/index.ts'
import { useToasts } from '../../toasts/store.ts'
import { updatesForReview, visibleUpdates } from '../lookup.ts'
import { useUpdates } from '../updates.ts'

type Details = Parameters<typeof updatesForReview>[2]

/** Updates SMAPI reports that the review does not offer, because the Nexus page they come from is hidden or removed. */
export function withheldUpdates(
  result: Parameters<typeof visibleUpdates>[0],
  profile: Profile | undefined,
  details: Details,
): Update[] {
  const offered = new Set(updatesForReview(result, profile, details).map((u) => `${u.key}/${u.id}`))
  return visibleUpdates(result, profile).filter((u) => !offered.has(`${u.key}/${u.id}`))
}

/** A review that was asked for but has nothing to show closes and says so, rather than doing nothing. */
export function useEmptyReviewNotice(empty: boolean) {
  useEffect(() => {
    if (empty) {
      useUpdates.getState().setReviewing(false)
      useToasts.getState().push({ kind: 'info', title: i18n._(msg`No updates to review`) })
    }
  }, [empty])
}
