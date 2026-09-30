import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField } from '@mui/material'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function NewProfileDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const create = useProfiles((s) => s.create)
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const close = () => {
    setName('')
    setError('')
    onClose()
  }
  const submit = async () => {
    const trimmed = name.trim()
    if (!trimmed) {
      setError(t`Enter a name for the profile.`)
      return
    }
    try {
      await create(trimmed)
      close()
    } catch (e) {
      setError(String(e))
    }
  }
  return (
    <Dialog
      open={open}
      onClose={close}
      slotProps={{ paper: { sx: { bgcolor: 'rgba(40,40,48,0.92)', minWidth: 360 } } }}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault()
          submit().catch(reportUnexpected)
        }}
      >
        <DialogTitle>{t`New profile`}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus={true}
            fullWidth={true}
            margin="dense"
            label={t`Profile name`}
            value={name}
            onChange={(e) => {
              setName(e.target.value)
              setError('')
            }}
            error={error !== ''}
            helperText={error || ' '}
            slotProps={{ htmlInput: { maxLength: 60 }, root: { sx: { userSelect: 'text' } } }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>{t`Cancel`}</Button>
          <Button type="submit" variant="contained">
            {t`Create`}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
