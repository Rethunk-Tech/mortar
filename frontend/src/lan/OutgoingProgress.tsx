import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { formatBytes } from '../i18n/bytes.ts'
import { useIncomingShares } from './incoming.ts'

// What a paired computer pulling a sent profile's files looks like from the sending side: the receiver's
// "Copying mod files" dialog, without the cancel the receiver owns.
export function OutgoingProgress() {
  const { t } = useLingui()
  const transfer = useIncomingShares((shares) =>
    Object.values(shares.outgoing)
      .filter((o) => !shares.hidden.includes(o.id))
      .sort((a, b) => b.id - a.id)
      .at(0),
  )
  const hide = useIncomingShares((shares) => shares.hideOutgoing)
  if (!transfer) {
    return null
  }
  const { peer, profile, current, total, bytes, totalBytes, state, reason } = transfer
  let text = t`Sending to ${peer}: ${current} of ${total} files`
  if (state === 'done') {
    text = t`${peer} has all the files for ${profile}`
  } else if (state !== 'sending') {
    const why = state === 'cancelled' ? t`${peer} cancelled it` : reason
    text = t`Sending to ${peer} stopped: ${why}`
  }
  const line =
    state === 'sending' && totalBytes > 0
      ? `${text} · ${formatBytes(bytes)} of ${formatBytes(totalBytes)}`
      : text
  return (
    <Dialog open={true} onClose={() => hide(transfer.id)}>
      <DialogTitle>{t`Copying mod files`}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {state === 'sending' && <CircularProgress size={24} />}
          <Typography>{line}</Typography>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => hide(transfer.id)}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
