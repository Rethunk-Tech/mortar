import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { useLaunch } from './store.ts'

export function StopDialog({
  open,
  game,
  onClose,
}: {
  open: boolean
  game: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const stop = useLaunch((s) => s.stop)
  return (
    <Dialog
      open={open}
      onClose={onClose}
      slotProps={{ paper: { sx: { bgcolor: 'rgba(40,40,48,0.92)', maxWidth: 420 } } }}
    >
      <DialogTitle>{t`Stop the game?`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`Stardew Valley will close now, and any progress since your last save is lost.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>{t`Cancel`}</Button>
        <Button
          color="error"
          variant="contained"
          sx={{ whiteSpace: 'nowrap' }}
          onClick={() => {
            onClose()
            stop(game)
          }}
        >
          {t`Stop game`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
