import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from '@mui/material'
import { type SyntheticEvent, useEffect, useState } from 'react'

/** One labelled TextField: Cancel and a contained confirm. Submit on Enter. */
export function PromptDialog({
  open,
  title,
  label,
  initial = '',
  placeholder,
  maxLength,
  confirmLabel,
  cancelLabel,
  allowEmpty = false,
  busy = false,
  onCancel,
  onSubmit,
}: {
  open: boolean
  title: string
  label: string
  initial?: string
  placeholder?: string
  maxLength?: number
  confirmLabel: string
  cancelLabel?: string
  allowEmpty?: boolean
  busy?: boolean
  onCancel: () => void
  onSubmit: (value: string) => void
}) {
  const { t } = useLingui()
  const [value, setValue] = useState(initial)
  useEffect(() => {
    if (open) {
      setValue(initial)
    }
  }, [open, initial])
  const trimmed = value.trim()
  const canSubmit = allowEmpty || trimmed !== ''
  const submit = (e: SyntheticEvent) => {
    e.preventDefault()
    if (busy || !canSubmit) {
      return
    }
    onSubmit(trimmed)
  }
  return (
    <Dialog open={open} onClose={busy ? undefined : onCancel}>
      <form onSubmit={submit}>
        <DialogTitle>{title}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus={true}
            fullWidth={true}
            label={label}
            placeholder={placeholder}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            slotProps={maxLength === undefined ? undefined : { htmlInput: { maxLength } }}
            disabled={busy}
          />
        </DialogContent>
        <DialogActions sx={{ flexWrap: 'wrap' }}>
          <Button onClick={onCancel} disabled={busy}>
            {cancelLabel ?? t`Cancel`}
          </Button>
          <Button type="submit" variant="contained" disabled={busy || !canSubmit}>
            {confirmLabel}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
