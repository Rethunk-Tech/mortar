import { Dialog, DialogContent } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Update } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useNow } from '../i18n/useNow.ts'
import { download } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { mergeCachedDetails } from './changelogRange.ts'
import { modId, updatesForReview } from './lookup.ts'
import { useNexusDetails } from './nexusDetails.ts'
import { paper } from './paper.ts'
import { useMods } from './store.ts'
import { DIALOG_WIDTH } from './updateReview/constants.ts'
import { EverywhereDialog } from './updateReview/EverywhereDialog.tsx'
import { loadAllDetails } from './updateReview/loadAll.ts'
import { PropagateUpdate } from './updateReview/PropagateUpdate.tsx'
import { ReviewFooter } from './updateReview/ReviewFooter.tsx'
import { ReviewList } from './updateReview/ReviewList.tsx'
import { ReviewTitle } from './updateReview/ReviewTitle.tsx'
import { UpdateBar as ReviewBar } from './updateReview/UpdateBar.tsx'
import { downloadable, installedCaution, pendingUpdate, updateWant } from './updateReview/wants.ts'
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
  const items = useQueue((s) => s.state.items)
  const mods = useMods((s) => s.mods)
  const [acked, setAcked] = useState<Record<string, boolean>>({})
  const now = useNow()
  const [propagating, setPropagating] = useState<Update[]>([])
  const [firstPropagating] = propagating
  const [include, setInclude] = useState<Record<string, boolean>>({})
  const [propagateAll, setPropagateAll] = useState(false)
  const [everywhereAll, setEverywhereAll] = useState(false)
  useEffect(() => {
    if (!open) {
      return
    }
    mergeCachedDetails(
      (updates?.updates ?? []).filter((u) => u.nexusId > 0).map((u) => u.nexusId),
    ).catch(reportUnexpected)
  }, [open, updates])
  const wanted = list
    .filter(
      (u) =>
        include[modId(u)] !== false &&
        downloadable(u) &&
        !pendingUpdate(items, profile.id, u) &&
        (installedCaution(mods, u) === '' || acked[modId(u)] === true),
    )
    .map(updateWant)
  const uncachedIds = [
    ...new Set(
      list.filter((u) => u.nexusId > 0 && !byId[u.nexusId]?.details).map((u) => u.nexusId),
    ),
  ]
  return (
    <Dialog
      open={open && list.length > 0}
      onClose={close}
      transitionDuration={0}
      maxWidth={false}
      slotProps={{
        paper: { sx: { ...paper.sx, width: DIALOG_WIDTH, maxWidth: 'calc(100% - 32px)' } },
      }}
    >
      <ReviewTitle
        count={list.length}
        profileName={profile.name}
        checkedLabel={checkedWithSmapi(checkedAt, now, updates?.unknown === true)}
        onClose={close}
      />
      <DialogContent sx={{ p: 0, borderTop: '1px solid var(--mortar-hairline-muted)' }}>
        <ReviewList
          list={list}
          profile={profile}
          mods={mods}
          acked={acked}
          include={include}
          onAck={(id, on) => setAcked((prev) => ({ ...prev, [id]: on }))}
          onInclude={(id, on) => setInclude((prev) => ({ ...prev, [id]: on }))}
        />
      </DialogContent>
      <ReviewFooter
        wantedCount={wanted.length}
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
          download(wanted, true)
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
      <EverywhereDialog
        open={everywhereAll}
        game={gameId}
        mods={list
          .filter((u) => wanted.some((w) => w.currentKey === u.key))
          .map((u) => ({ id: u.uniqueId, newKey: 'latest' }))}
        onClose={() => setEverywhereAll(false)}
      />
    </Dialog>
  )
}
