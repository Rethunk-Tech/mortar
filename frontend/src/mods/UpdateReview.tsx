import { Dialog, DialogContent } from '@mui/material'
import { memo, useEffect, useState } from 'react'
import type { Update } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useNowValue } from '../i18n/useNow.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { modId, updatesForReview } from './lookup.ts'
import { isFull, mergeCachedDetails, useNexusDetails } from './nexusDetails.ts'
import { useOptionalSkips } from './optionalFiles.ts'
import { useMods } from './store.ts'
import { DIALOG_WIDTH } from './updateReview/constants.ts'
import { EverywhereDialog } from './updateReview/EverywhereDialog.tsx'
import { KeptGroup } from './updateReview/KeptGroup.tsx'
import { keptUpdates } from './updateReview/kept.ts'
import { loadAllDetails } from './updateReview/loadAll.ts'
import { NeedsChoice } from './updateReview/NeedsChoice.tsx'
import { ReviewFooter } from './updateReview/ReviewFooter.tsx'
import { ReviewTitle } from './updateReview/ReviewTitle.tsx'
import { SourceGroups } from './updateReview/SourceGroups.tsx'
import { UndoAllConfirm } from './updateReview/UndoAllConfirm.tsx'
import {
  listedBySource,
  needChoiceUpdates,
  sameSourceUpdates,
  updateAll,
} from './updateReview/updateAll.ts'
import { WithheldGroup } from './updateReview/WithheldGroup.tsx'
import { installedCaution, pendingUpdate, withOptional } from './updateReview/wants.ts'
import { useEmptyReviewNotice, withheldUpdates } from './updateReview/withheld.ts'
import { checkedLabel, useUpdates } from './updates.ts'

function useMergeCachedDetails(open: boolean) {
  const updates = useUpdates((s) => s.updates)
  useEffect(() => {
    if (!open) {
      return
    }
    mergeCachedDetails(
      (updates?.updates ?? []).filter((u) => u.nexusId > 0).map((u) => u.nexusId),
    ).catch(reportUnexpected)
  }, [open, updates])
}

function uncachedNexusIds(
  list: Update[],
  byId: ReturnType<typeof useNexusDetails.getState>['byId'],
) {
  return [
    ...new Set(list.filter((u) => u.nexusId > 0 && !isFull(byId[u.nexusId])).map((u) => u.nexusId)),
  ]
}

// Mounted from the first time the review opens (then kept, so it can fade out): until then a save of the profile
// does not rebuild a closed review.
const UpdateReview = memo(function UpdateReviewGate() {
  const open = useUpdates((s) => s.reviewing)
  const [opened, setOpened] = useState(open)
  if (open && !opened) {
    setOpened(true)
  }
  return opened ? <OpenProfileReview /> : null
})

function OpenProfileReview() {
  const profile = useProfiles(openProfileOf)
  return profile ? <UpdateReviewDialog profile={profile} /> : null
}

function UpdateReviewDialog({ profile }: { profile: Profile }) {
  const signedIn = useNexus((s) => s.signedIn)
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const open = useUpdates((s) => s.reviewing)
  const updates = useUpdates((s) => s.updates)
  const checkedAt = useUpdates((s) => s.checkedAt)
  const unknown = updates?.unknown === true
  const checked = useNowValue((now) => checkedLabel(checkedAt, now, unknown))
  const setReviewing = useUpdates((s) => s.setReviewing)
  const close = () => setReviewing(false)
  const byId = useNexusDetails((s) => s.byId)
  const list = updatesForReview(updates, profile, byId)
  const withheld = withheldUpdates(updates, profile, byId)
  const items = useQueue((s) => s.state.items)
  const mods = useMods((s) => s.mods)
  const [acked, setAcked] = useState<Record<string, boolean>>({})
  const [everywhereAll, setEverywhereAll] = useState(false)
  const [loadingAll, setLoadingAll] = useState(false)
  const skipped = useOptionalSkips((s) => s.skipped)
  useMergeCachedDetails(open)
  useEmptyReviewNotice(open && updates !== null && list.length === 0 && withheld.length === 0)
  const needChoice = needChoiceUpdates(list)
  const chosen = sameSourceUpdates(list).filter(
    (u) =>
      !pendingUpdate(items, profile.id, u) &&
      (installedCaution(u) === '' || acked[modId(u)] === true),
  )
  const wanted = chosen.flatMap((u) =>
    withOptional(u, profile, byId[u.nexusId]?.details?.files ?? [], skipped[u.key] === true),
  )
  const onAck = (id: string, on: boolean) => setAcked((prev) => ({ ...prev, [id]: on }))
  const uncachedIds = uncachedNexusIds(list, byId)
  return (
    <Dialog
      open={open && (list.length > 0 || withheld.length > 0)}
      onClose={close}
      maxWidth={false}
      slotProps={{
        paper: { sx: { width: DIALOG_WIDTH, maxWidth: 'calc(100% - 32px)' } },
      }}
    >
      <ReviewTitle
        count={list.length}
        profileName={profile.name}
        checkedLabel={checked}
        canLoadAll={signedIn && uncachedIds.length > 0}
        loadingAll={loadingAll}
        onLoadAll={() => {
          loadAllDetails(uncachedIds, setLoadingAll).then(
            () => undefined,
            () => undefined,
          )
        }}
      />
      <DialogContent sx={{ p: 0, borderTop: '1px solid var(--mortar-hairline-muted)' }}>
        <SourceGroups list={listedBySource(list)} profile={profile} acked={acked} onAck={onAck} />
        <NeedsChoice list={needChoice} profile={profile} acked={acked} onAck={onAck} />
        <WithheldGroup withheld={withheld} mods={mods} />
        <KeptGroup kept={keptUpdates(updates, profile)} mods={mods} />
      </DialogContent>
      <ReviewFooter
        wantedCount={chosen.length}
        onClose={close}
        onUpdate={() => {
          close()
          updateAll(gameId, profile.id, wanted, needChoice.length).catch(reportUnexpected)
        }}
        onEverywhere={() => setEverywhereAll(true)}
      />
      <UndoAllConfirm />
      <EverywhereDialog
        open={everywhereAll}
        game={gameId}
        mods={list
          .filter((u) => wanted.some((w) => w.currentKey === u.key))
          .map((u) => ({ id: u.id, newKey: 'latest' }))}
        onClose={() => setEverywhereAll(false)}
      />
    </Dialog>
  )
}

export { UpdateReview }
