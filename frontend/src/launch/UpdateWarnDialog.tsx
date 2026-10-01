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

export function UpdateWarnDialog() {
  const { t } = useLingui()
  const warn = useLaunch((s) => s.updateWarn)
  const cancel = useLaunch((s) => s.dismissUpdateWarn)
  const playAnyway = useLaunch((s) => s.playAnyway)
  const openProblems = useLaunch((s) => s.openProblems)
  return (
    <Dialog
      open={warn !== null}
      onClose={cancel}
      transitionDuration={0}
      slotProps={{ paper: { sx: { maxWidth: 480 } } }}
    >
      <DialogTitle>{t`The game was updated`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`Stardew Valley is now ${warn?.installed ?? ''}. This profile last launched on ${warn?.recorded ?? ''}.`}
        </DialogContentText>
        {(warn?.broken.length ?? 0) > 0 ? (
          <List dense={true} sx={{ mt: 1 }}>
            {warn?.broken.map((mod) => (
              <ListItem key={`${mod.key}/${mod.uniqueId}`} disableGutters={true}>
                <ListItemText
                  primary={mod.name}
                  secondary={mod.brokeIn ? t`Broke in ${mod.brokeIn}` : t`Broken for this version`}
                />
              </ListItem>
            ))}
          </List>
        ) : (
          <DialogContentText sx={{ mt: 1 }}>
            {t`None of this profile's mods are marked broken for the new version.`}
          </DialogContentText>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={cancel} sx={{ whiteSpace: 'nowrap' }}>{t`Cancel`}</Button>
        <Button onClick={openProblems} sx={{ whiteSpace: 'nowrap' }}>{t`Open problems`}</Button>
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
