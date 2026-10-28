import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ArrowUpCircle, ShieldCheck } from 'lucide-react'
import { memo, useEffect, useState } from 'react'
import type { Update } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useNowValue } from '../i18n/useNow.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { modId, updatesForReview } from './lookup.ts'
import type { NexusById } from './nexusDetails.ts'
import { isFull, mergeCachedDetails, useGameNexusDetails } from './nexusDetails.ts'
import { useOptionalSkips } from './optionalFiles.ts'
import { useMods } from './store.ts'
import { EverywhereDialog } from './updateReview/EverywhereDialog.tsx'
import { KeptGroup } from './updateReview/KeptGroup.tsx'
import { keptUpdates } from './updateReview/kept.ts'
import { loadAllDetails } from './updateReview/loadAll.ts'
import { NeedsChoice } from './updateReview/NeedsChoice.tsx'
import { SourceGroups } from './updateReview/SourceGroups.tsx'
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
  const game = useProfiles((s) => s.game?.id ?? '')
  const updates = useUpdates((s) => s.updates)
  useEffect(() => {
    if (!open) {
      return
    }
    mergeCachedDetails(
      game,
      (updates?.updates ?? []).filter((u) => u.nexusId > 0).map((u) => u.nexusId),
    ).catch(reportUnexpected)
  }, [open, game, updates])
}

function uncachedNexusIds(list: Update[], byId: NexusById) {
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
  const { t } = useLingui()
  const signedIn = useNexus((s) => s.signedIn)
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const open = useUpdates((s) => s.reviewing)
  const updates = useUpdates((s) => s.updates)
  const checkedAt = useUpdates((s) => s.checkedAt)
  const unknown = updates?.unknown === true
  const checked = useNowValue((now) => checkedLabel(checkedAt, now, unknown))
  const setReviewing = useUpdates((s) => s.setReviewing)
  const close = () => setReviewing(false)
  const byId = useGameNexusDetails(gameId)
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
  const count = list.length
  const wantedCount = chosen.length
  return (
    <>
      <AppDialog
        open={open && (list.length > 0 || withheld.length > 0)}
        onClose={close}
        size="lg"
        title={plural(count, {
          one: `# update for ${{ profile: profile.name }}`,
          other: `# updates for ${{ profile: profile.name }}`,
        })}
        subtitle={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <span>{checked}</span>
            {signedIn && uncachedIds.length > 0 ? (
              <Button
                size="small"
                disabled={loadingAll}
                onClick={() => {
                  loadAllDetails(gameId, uncachedIds, setLoadingAll).then(
                    () => undefined,
                    () => undefined,
                  )
                }}
              >
                {loadingAll ? t`Loading changes…` : t`Load all changes`}
              </Button>
            ) : null}
          </Box>
        }
        icon={
          <ToneIcon tone="neutral">
            <ArrowUpCircle size={22} />
          </ToneIcon>
        }
        footer={
          <DialogFooter
            aside={
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                <ShieldCheck size={16} aria-hidden={true} style={{ flexShrink: 0 }} />
                <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'inherit' }}>
                  {t`Settings are kept and saves backed up.`}
                </Typography>
              </Box>
            }
          >
            <Button onClick={close}>{t`Close`}</Button>
            {chosen.length > 0 ? (
              <Button onClick={() => setEverywhereAll(true)}>{t`Update in every profile…`}</Button>
            ) : null}
            {chosen.length > 0 ? (
              <Button
                variant="contained"
                onClick={() => {
                  close()
                  updateAll(gameId, profile.id, wanted, needChoice.length).catch(reportUnexpected)
                }}
              >
                {t`Update all (${wantedCount})`}
              </Button>
            ) : null}
          </DialogFooter>
        }
      >
        {/* The groups run to the dialog's edges; the body's padding is taken back. */}
        <Box sx={{ mx: -3, my: -2.5 }}>
          <SourceGroups list={listedBySource(list)} profile={profile} acked={acked} onAck={onAck} />
          <NeedsChoice list={needChoice} profile={profile} acked={acked} onAck={onAck} />
          <WithheldGroup withheld={withheld} mods={mods} />
          <KeptGroup kept={keptUpdates(updates, profile)} mods={mods} />
        </Box>
      </AppDialog>
      <EverywhereDialog
        open={everywhereAll}
        game={gameId}
        mods={list
          .filter((u) => wanted.some((w) => w.currentKey === u.key))
          .map((u) => ({ id: u.id, newKey: 'latest' }))}
        onClose={() => setEverywhereAll(false)}
      />
    </>
  )
}

export { UpdateReview }
