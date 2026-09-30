import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'

export function SmapiWarnDialog({
  open,
  onClose,
  onPlay,
}: {
  open: boolean
  onClose: () => void
  onPlay: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog
      open={open}
      onClose={onClose}
      transitionDuration={0}
      slotProps={{ paper: { sx: { bgcolor: 'rgb(40,40,48)', maxWidth: 440 } } }}
    >
      <DialogTitle>{t`Steam will still start SMAPI`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`Stardew Valley's Steam launch options run SMAPI in place of the game. Steam will still start SMAPI unless that launch option is removed. Mortar will not change it.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>
          {t`Cancel`}
        </Button>
        <Button variant="contained" sx={{ whiteSpace: 'nowrap' }} onClick={onPlay}>
          {t`Play without mods`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
