import { Dialog, DialogContent } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Update } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useNow } from '../i18n/useNow.ts'
import { useProfiles } from '../profiles/store.ts'
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
import { PropagateUpdate } from './updateReview/PropagateUpdate.tsx'
import { ReviewFooter } from './updateReview/ReviewFooter.tsx'
import { ReviewTitle } from './updateReview/ReviewTitle.tsx'
import { SourceGroups } from './updateReview/SourceGroups.tsx'
import { UndoAllConfirm } from './updateReview/UndoAllConfirm.tsx'
import { UpdateBar as ReviewBar } from './updateReview/UpdateBar.tsx'
import { needChoiceUpdates, sameSourceUpdates, updateAll } from './updateReview/updateAll.ts'
import { WithheldGroup } from './updateReview/WithheldGroup.tsx'
import { installedCaution, pendingUpdate, withOptional } from './updateReview/wants.ts'
import { useEmptyReviewNotice, withheldUpdates } from './updateReview/withheld.ts'
import { checkedWithSmapi, useUpdates } from './updates.ts'

export function UpdateBar() {
  return <ReviewBar />
}

export function UpdateReview({ profile }: { profile: Profile }) {
  const signedIn = useNexus((s) => s.signedIn)
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const open = useUpdates((s) => s.reviewing)
  const updates = useUpdates((s) => s.updates)
  const checkedAt = useUpdates((s) => s.checkedAt)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const close = () => setReviewing(false)
  const byId = useNexusDetails((s) => s.byId)
  const list = updatesForReview(updates, profile, byId)
  const withheld = withheldUpdates(updates, profile, byId)
  const items = useQueue((s) => s.state.items)
  const mods = useMods((s) => s.mods)
  const [acked, setAcked] = useState<Record<string, boolean>>({})
  const now = useNow()
  const [propagating, setPropagating] = useState<Update[]>([])
  const [firstPropagating] = propagating
  const [include, setInclude] = useState<Record<string, boolean>>({})
  const [propagateAll, setPropagateAll] = useState(false)
  const [everywhereAll, setEverywhereAll] = useState(false)
  const [loadingAll, setLoadingAll] = useState(false)
  const skipped = useOptionalSkips((s) => s.skipped)
  useEffect(() => {
    if (!open) {
      return
    }
    mergeCachedDetails(
      (updates?.updates ?? []).filter((u) => u.nexusId > 0).map((u) => u.nexusId),
    ).catch(reportUnexpected)
  }, [open, updates])
  useEmptyReviewNotice(open && updates !== null && list.length === 0 && withheld.length === 0)
  const needChoice = needChoiceUpdates(list)
  const sameSource = list.filter((u) => !u.switch)
  const chosen = sameSourceUpdates(list).filter(
    (u) =>
      include[modId(u)] !== false &&
      !pendingUpdate(items, profile.id, u) &&
      (installedCaution(mods, u) === '' || acked[modId(u)] === true),
  )
  const wanted = chosen.flatMap((u) =>
    withOptional(u, profile, byId[u.nexusId]?.details?.files ?? [], skipped[u.key] === true),
  )
  const onAck = (id: string, on: boolean) => setAcked((prev) => ({ ...prev, [id]: on }))
  const onInclude = (id: string, on: boolean) => setInclude((prev) => ({ ...prev, [id]: on }))
  const uncachedIds = [
    ...new Set(list.filter((u) => u.nexusId > 0 && !isFull(byId[u.nexusId])).map((u) => u.nexusId)),
  ]
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
        checkedLabel={checkedWithSmapi(checkedAt, now, updates?.unknown === true)}
        onClose={close}
      />
      <DialogContent sx={{ p: 0, borderTop: '1px solid var(--mortar-hairline-muted)' }}>
        <SourceGroups
          list={sameSource}
          profile={profile}
          mods={mods}
          acked={acked}
          include={include}
          onAck={onAck}
          onInclude={onInclude}
        />
        <NeedsChoice
          list={needChoice}
          profile={profile}
          mods={mods}
          acked={acked}
          include={include}
          onAck={onAck}
          onInclude={onInclude}
        />
        <WithheldGroup withheld={withheld} mods={mods} />
        <KeptGroup kept={keptUpdates(updates, profile)} mods={mods} />
      </DialogContent>
      <ReviewFooter
        wantedCount={chosen.length}
        signedIn={signedIn}
        uncachedIds={uncachedIds}
        loadingAll={loadingAll}
        propagateAll={propagateAll}
        onClose={close}
        onLoadAll={() => {
          loadAllDetails(uncachedIds, setLoadingAll).then(
            () => undefined,
            () => undefined,
          )
        }}
        onPropagate={setPropagateAll}
        onUpdate={() => {
          close()
          updateAll(gameId, profile.id, wanted, needChoice.length)
            .then((added) => {
              if (added && propagateAll) {
                setPropagating(list.filter((u) => wanted.some((w) => w.currentKey === u.key)))
              }
            })
            .catch(reportUnexpected)
        }}
        onEverywhere={() => setEverywhereAll(true)}
      />
      {firstPropagating ? (
        <PropagateUpdate
          profile={profile}
          update={firstPropagating}
          onDone={() => setPropagating((pending) => pending.slice(1))}
        />
      ) : null}
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
