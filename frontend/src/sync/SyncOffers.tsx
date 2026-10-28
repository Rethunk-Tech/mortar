import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import type { Offer } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/models.ts'
import {
  Diff as ReadDiff,
  Resolve,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/service.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { diffIsStale, revisionToAnswer, type ShownDiff } from './revision.ts'
import { useSync } from './store.ts'

function OfferRow({ offer }: { offer: Offer }) {
  const { t } = useLingui()
  const [shown, setShown] = useState<ShownDiff | null>(null)
  const [updated, setUpdated] = useState(false)
  const [busy, run] = usePending()
  const diff = shown?.diff
  const { game, remote, revision } = offer
  const load = useCallback(
    () =>
      ReadDiff(game, remote)
        .then((d) => setShown({ diff: d, revision }))
        .catch(reportUnexpected),
    [game, remote, revision],
  )
  // The other machine wrote again while the diff was open: show the new one rather than answer an old one.
  useEffect(() => {
    if (diffIsStale(offer.revision, shown)) {
      setUpdated(true)
      load()
    }
  }, [offer.revision, shown, load])
  const answer = (choice: string) =>
    run(() => Resolve(offer.game, offer.remote, revisionToAnswer(offer.revision, shown), choice), {
      quietExpected: true,
    })
  return (
    <Box sx={{ py: space.gap, borderBottom: '1px solid var(--mortar-hairline)' }}>
      <Typography sx={{ fontWeight: 600 }}>
        {offer.conflict
          ? t`${{ name: offer.name }} was changed on ${{ machine: offer.machine }} and on this machine`
          : t`${{ name: offer.name }} was changed on ${{ machine: offer.machine }}`}
      </Typography>
      {offer.conflict ? (
        <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
          {t`Both sides changed it, so Mortar will not merge them. Pick one side.`}
        </Typography>
      ) : null}
      <Box sx={{ display: 'flex', gap: space.gap, mt: space.gap, flexWrap: 'wrap' }}>
        <Button variant="contained" disabled={busy} onClick={() => answer('theirs')}>
          {offer.conflict ? t`Use theirs` : t`Apply`}
        </Button>
        {offer.new ? null : (
          <Button disabled={busy} onClick={() => answer('mine')}>{t`Keep mine`}</Button>
        )}
        <Button
          disabled={busy}
          onClick={() => {
            setUpdated(false)
            load()
          }}
        >{t`Show diff`}</Button>
      </Box>
      {diff ? (
        <Box sx={{ mt: space.gap, fontSize: 'base.fontSize' }}>
          {updated ? (
            <Box
              sx={{ color: 'text.secondary', mb: 0.5 }}
            >{t`Updated from ${{ machine: offer.machine }}`}</Box>
          ) : null}
          {(diff.add ?? []).map((name) => (
            <Box key={`add-${name}`}>{t`Adds ${name}`}</Box>
          ))}
          {(diff.remove ?? []).map((name) => (
            <Box key={`remove-${name}`}>{t`Removes ${name}`}</Box>
          ))}
          {(diff.add?.length ?? 0) + (diff.remove?.length ?? 0) === 0 ? (
            <Box>{t`Only settings and options differ.`}</Box>
          ) : null}
        </Box>
      ) : null}
    </Box>
  )
}

// The profiles another machine changed through the sync folder, each answered by taking its version or keeping this one.
export function SyncOffers() {
  const { t } = useLingui()
  const offers = useSync((s) => s.offers)
  const stalled = useSync((s) => s.stalled)
  const open = useSync((s) => s.open)
  const setOpen = useSync((s) => s.setOpen)
  return (
    <AppDialog
      open={open && offers.length + stalled.length > 0}
      onClose={() => setOpen(false)}
      title={t`Changes from your other machines`}
      footer={
        <DialogFooter>
          <Button onClick={() => setOpen(false)}>{t`Close`}</Button>
        </DialogFooter>
      }
    >
      {offers.map((o) => (
        <OfferRow key={`${o.game}/${o.remote}`} offer={o} />
      ))}
      {stalled.map((st) => (
        <Box
          key={`stalled-${st.game}/${st.remote}`}
          sx={{ py: space.gap, borderBottom: '1px solid var(--mortar-hairline)' }}
        >
          <Typography
            sx={{ fontWeight: 600 }}
          >{t`${{ name: st.name }} is not syncing yet`}</Typography>
          <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
            {t`Waiting for ${{ machine: st.machine }}'s profile file to finish syncing. Changes made here are written out once it arrives.`}
          </Typography>
        </Box>
      ))}
    </AppDialog>
  )
}
