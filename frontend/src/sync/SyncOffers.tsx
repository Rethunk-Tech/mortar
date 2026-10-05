import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import type {
  Diff,
  Offer,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/models.ts'
import {
  Diff as ReadDiff,
  Resolve,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/syncsvc/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useSync } from './store.ts'

function OfferRow({ offer }: { offer: Offer }) {
  const { t } = useLingui()
  const [diff, setDiff] = useState<Diff | null>(null)
  const [busy, setBusy] = useState(false)
  const answer = (choice: string) => {
    setBusy(true)
    Resolve(offer.game, offer.remote, choice)
      .catch(reportUnexpected)
      .finally(() => setBusy(false))
  }
  return (
    <Box sx={{ py: 1.25, borderBottom: '1px solid var(--mortar-hairline)' }}>
      <Typography sx={{ fontWeight: 600 }}>
        {offer.conflict
          ? t`${offer.name} was changed on ${offer.machine} and on this machine`
          : t`${offer.name} was changed on ${offer.machine}`}
      </Typography>
      {offer.conflict ? (
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
          {t`Both sides changed it, so Mortar will not merge them. Pick one side.`}
        </Typography>
      ) : null}
      <Box sx={{ display: 'flex', gap: 1, mt: 1, flexWrap: 'wrap' }}>
        <Button variant="contained" disabled={busy} onClick={() => answer('theirs')}>
          {offer.conflict ? t`Use theirs` : t`Apply`}
        </Button>
        {offer.new ? null : (
          <Button disabled={busy} onClick={() => answer('mine')}>{t`Keep mine`}</Button>
        )}
        <Button
          disabled={busy}
          onClick={() => ReadDiff(offer.game, offer.remote).then(setDiff).catch(reportUnexpected)}
        >{t`Show diff`}</Button>
      </Box>
      {diff ? (
        <Box sx={{ mt: 1, fontSize: 13 }}>
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
  const open = useSync((s) => s.open)
  const setOpen = useSync((s) => s.setOpen)
  return (
    <Dialog open={open && offers.length > 0} onClose={() => setOpen(false)} fullWidth={true}>
      <DialogTitle>{t`Changes from your other machines`}</DialogTitle>
      <DialogContent>
        {offers.map((o) => (
          <OfferRow key={`${o.game}/${o.remote}`} offer={o} />
        ))}
      </DialogContent>
      <DialogActions>
        <Button onClick={() => setOpen(false)}>{t`Later`}</Button>
      </DialogActions>
    </Dialog>
  )
}
