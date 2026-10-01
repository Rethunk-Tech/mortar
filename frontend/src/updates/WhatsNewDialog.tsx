import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { WhatsNew } from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/models.ts'
import {
  AckWhatsNew,
  WhatsNew as LoadWhatsNew,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/service.ts'

export function WhatsNewDialog() {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const [payload, setPayload] = useState<WhatsNew | null>(null)
  useEffect(() => {
    LoadWhatsNew()
      .then((w) => {
        if (w.notes && w.version) {
          setPayload(w)
          setOpen(true)
        }
      })
      .catch(() => undefined)
  }, [])
  const close = () => {
    setOpen(false)
    AckWhatsNew().catch(() => undefined)
  }
  return (
    <Dialog
      open={open}
      onClose={close}
      transitionDuration={0}
      slotProps={{ paper: { sx: { maxWidth: 520 } } }}
    >
      <DialogTitle>{t`What's new in Mortar ${payload?.version ?? ''}`}</DialogTitle>
      <DialogContent>
        <DialogContentText
          component="div"
          sx={{ whiteSpace: 'pre-wrap', fontSize: 14, color: 'rgba(225,225,230,0.95)' }}
        >
          {payload?.notes}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button variant="contained" onClick={close} sx={{ whiteSpace: 'nowrap' }}>
          {t`Got it`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
