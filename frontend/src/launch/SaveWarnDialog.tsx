import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  List,
  ListItem,
  ListItemText,
} from '@mui/material'
import { useLaunch } from './store.ts'

// Before Play: the newest save uses mods this profile lacks or has switched off, so loading it may break it.
export function SaveWarnDialog() {
  const { t } = useLingui()
  const warn = useLaunch((s) => s.saveWarn)
  const cancel = useLaunch((s) => s.dismissSaveWarn)
  const playAnyway = useLaunch((s) => s.playAnyway)
  const openSaves = useLaunch((s) => s.openSaves)
  const save = warn?.save
  return (
    <Dialog
      open={warn !== null}
      onClose={cancel}
      transitionDuration={0}
      slotProps={{ paper: { sx: { maxWidth: 480 } } }}
    >
      <DialogTitle>{t`Your last save needs other mods`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`${save?.farmer ?? ''}'s farm (${save?.folder ?? ''}) was last played with mods this profile does not have on. Loading it without them can lose their items or break the save.`}
        </DialogContentText>
        <List dense={true} sx={{ mt: 1 }}>
          {(save?.missing ?? []).map((m) => (
            <ListItem key={m.uniqueId} disableGutters={true}>
              <ListItemText
                primary={m.name || m.uniqueId}
                secondary={m.disabled ? t`Switched off in this profile` : t`Not in this profile`}
              />
            </ListItem>
          ))}
        </List>
      </DialogContent>
      <DialogActions>
        <Button onClick={cancel} sx={{ whiteSpace: 'nowrap' }}>{t`Cancel`}</Button>
        <Button onClick={openSaves} sx={{ whiteSpace: 'nowrap' }}>{t`Open saves`}</Button>
        <Button
          variant="contained"
          onClick={() => playAnyway().catch(() => undefined)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Play anyway`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
