import { useLingui } from '@lingui/react/macro'
import { Button, List, ListItem, ListItemText, Typography } from '@mui/material'
import { localId } from '../mods/dependents.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { cancelling } from './playModeState.ts'
import { useLaunch } from './store.ts'

export function SaveWarnDialog() {
  const { t } = useLingui()
  const warn = useLaunch((s) => s.saveWarn)
  const cancel = useLaunch((s) => s.dismissSaveWarn)
  const quit = cancelling(cancel)
  const playAnyway = useLaunch((s) => s.playAnyway)
  const save = warn?.save
  const missing = save?.missing ?? []
  return (
    <AppDialog
      open={warn !== null}
      onClose={quit}
      size="sm"
      title={t`Your last save needs other mods`}
      footer={
        <DialogFooter>
          <Button onClick={quit}>{t`Cancel`}</Button>
          <Button onClick={useLaunch.getState().openSaves}>{t`Open saves`}</Button>
          <Button variant="contained" onClick={() => playAnyway().catch(reportUnexpected)}>
            {t`Play anyway`}
          </Button>
        </DialogFooter>
      }
    >
      <Typography sx={{ color: 'text.secondary' }}>
        {t`${{ farmer: save?.farmer ?? '' }}'s farm (${{ path: save?.folder ?? '' }}) was last played with mods this profile lacks or has disabled. Playing without them can lose their items or break the save.`}
      </Typography>
      <List dense={true} sx={{ mt: 1 }}>
        {missing.map((m) => (
          <ListItem key={m.id} disableGutters={true}>
            <ListItemText
              primary={
                <span title={(m.name ?? '').trim() === '' ? localId(m.id) : undefined}>
                  {(m.name ?? '').trim() === '' ? t`Unknown mod` : m.name}
                </span>
              }
              secondary={m.disabled ? t`Disabled in this profile` : t`Not in this profile`}
            />
          </ListItem>
        ))}
      </List>
    </AppDialog>
  )
}
