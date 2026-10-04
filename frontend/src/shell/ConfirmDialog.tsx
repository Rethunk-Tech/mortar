import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import type { ReactNode } from 'react'

/** A yes/no question: title, one paragraph (or custom content), Cancel and one confirm button. */
export function ConfirmDialog({
  open,
  title,
  body,
  children,
  confirmLabel,
  cancelLabel,
  color = 'primary',
  busy = false,
  maxWidth = 440,
  onCancel,
  onConfirm,
}: {
  open: boolean
  title: ReactNode
  body?: ReactNode
  children?: ReactNode
  confirmLabel: ReactNode
  cancelLabel?: ReactNode
  color?: 'primary' | 'error' | 'warning'
  busy?: boolean
  maxWidth?: number
  onCancel: () => void
  onConfirm: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog
      open={open}
      onClose={busy ? undefined : onCancel}
      slotProps={{ paper: { sx: { maxWidth } } }}
    >
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        {body === undefined ? null : <DialogContentText>{body}</DialogContentText>}
        {children}
      </DialogContent>
      <DialogActions sx={{ flexWrap: 'wrap' }}>
        <Button onClick={onCancel} disabled={busy} autoFocus={color === 'error'}>
          {cancelLabel ?? t`Cancel`}
        </Button>
        <Button
          variant="contained"
          color={color}
          disabled={busy}
          autoFocus={color !== 'error'}
          onClick={onConfirm}
        >
          {confirmLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
