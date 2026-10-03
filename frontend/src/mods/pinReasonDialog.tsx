import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from '@mui/material'
import { useState } from 'react'
import { paper } from './paper.ts'
import { usePinReasonDialog } from './pinReasonStore.ts'
import { pinMany, pinMod } from './storeEntries.ts'

export function PinReasonDialog() {
  const { t } = useLingui()
  const target = usePinReasonDialog((s) => s.target)
  const close = usePinReasonDialog((s) => s.close)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const open = target !== null
  const confirm = async () => {
    if (!target) {
      return
    }
    setBusy(true)
    try {
      if (target.mods.length === 1) {
        await pinMod(target.mods[0], true, reason)
      } else {
        await pinMany(target.mods, true, reason)
      }
      close()
      setReason('')
    } finally {
      setBusy(false)
    }
  }
  const name = target?.mods[0]?.name ?? ''
  const many = (target?.mods.length ?? 0) > 1
  return (
    <Dialog
      open={open}
      onClose={() => !busy && close()}
      transitionDuration={0}
      slotProps={{
        paper: { ...paper, sx: { ...paper.sx, width: 420, maxWidth: 'calc(100% - 48px)' } },
      }}
    >
      <DialogTitle sx={{ fontWeight: 700 }}>
        {many ? t`Pin selected versions` : t`Pin ${name}`}
      </DialogTitle>
      <DialogContent>
        <TextField
          autoFocus={true}
          fullWidth={true}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder={t`Reason (optional)`}
          slotProps={{ htmlInput: { 'aria-label': t`Reason (optional)` } }}
        />
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button onClick={close} disabled={busy}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          onClick={() => confirm().catch(() => undefined)}
          disabled={busy}
        >
          {t`Pin`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
