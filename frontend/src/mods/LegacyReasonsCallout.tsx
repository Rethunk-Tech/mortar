import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { memo, useState } from 'react'
import { SetIntentOffered } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useCustomCategories } from './customCategories.ts'
import { customCategoryById, resolvedCategoryLabel } from './group.ts'
import { LegacyReview } from './LegacyReview.tsx'
import { ListCallout } from './ListCallout.tsx'
import { legacyImplicit } from './legacyReasons.ts'
import { markImplicit } from './markImplicit.ts'
import { siteCategory } from './modSearch.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { useMods } from './store.ts'

const NONE: string[] = []

// Once per profile from before install reasons: offers to mark the libraries and tilesheet packs its game's defaults
// name as implicit. Either answer ends the offer.
export const LegacyReasonsCallout = memo(function LegacyReasonsCalloutView() {
  const offered = useProfiles((s) => openProfileOf(s)?.intentOffered ?? true)
  return offered ? null : <LegacyReasonsOffer />
})

function LegacyReasonsOffer() {
  const { t } = useLingui()
  const profile = useProfiles(openProfileOf)
  const mods = useMods((s) => s.mods)
  const game = useProfiles((s) => s.game?.id ?? '')
  const nexusById = useNexusDetails((s) => s.byId)
  const customById = customCategoryById(useCustomCategories((s) => s.categories))
  const supporting = useMods((s) => s.problems?.supporting ?? NONE)
  const [pending, run] = usePending()
  const [reviewing, setReviewing] = useState(false)
  if (!profile || profile.intentOffered) {
    return null
  }
  const picks = legacyImplicit(
    profile,
    mods,
    (e) =>
      resolvedCategoryLabel(
        e.categoryOverride,
        siteCategory(e, nexusById[e.source.modId ?? 0]?.details?.category),
        customById,
      ),
    supporting,
  )
  if (picks.length === 0) {
    return null
  }
  const answer = (keys: string[]) =>
    run(
      async () => {
        if (keys.length > 0) {
          await markImplicit(
            mods.filter((m) => keys.includes(m.key)),
            true,
          )
        }
        setReviewing(false)
        useProfiles.getState().replace(await SetIntentOffered(game, profile.id))
      },
      { errorTitle: t`Could not save that answer` },
    )
  return (
    <ListCallout
      text={plural(picks.length, {
        one: 'Mortar identified # mod as a library or supporting mod',
        other: 'Mortar identified # mods as libraries or supporting mods',
      })}
      actions={
        <>
          <Button size="small" variant="text" disabled={pending} onClick={() => answer([])}>
            {t`Keep explicit`}
          </Button>
          <Button size="small" variant="text" disabled={pending} onClick={() => setReviewing(true)}>
            {t`Review…`}
          </Button>
          <Button
            size="small"
            variant="contained"
            disabled={pending}
            onClick={() => answer(picks.map((p) => p.key))}
          >
            {t`Mark implicit`}
          </Button>
          {reviewing ? (
            <LegacyReview
              picks={picks}
              mods={mods}
              busy={pending}
              onCancel={() => setReviewing(false)}
              onMark={answer}
            />
          ) : null}
        </>
      }
    />
  )
}
