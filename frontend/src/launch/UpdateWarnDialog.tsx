import { useLingui } from '@lingui/react/macro'
import { Button, List, ListItem, ListItemText, Typography } from '@mui/material'
import { useGameName } from '../games/info.ts'
import { modId } from '../mods/lookup.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { cancelling } from './playModeState.ts'
import { useLaunch } from './store.ts'

export function UpdateWarnDialog() {
  const { t } = useLingui()
  const gameName = useGameName()
  const warn = useLaunch((s) => s.updateWarn)
  const cancel = cancelling(useLaunch((s) => s.dismissUpdateWarn))
  const playAnyway = useLaunch((s) => s.playAnyway)
  const openProblems = useLaunch((s) => s.openProblems)
  return (
    <AppDialog
      open={warn !== null}
      onClose={cancel}
      size="sm"
      title={t`The game was updated`}
      footer={
        <DialogFooter>
          <Button onClick={cancel}>{t`Cancel`}</Button>
          <Button onClick={openProblems}>{t`Open problems`}</Button>
          <Button variant="contained" onClick={() => playAnyway().catch(reportUnexpected)}>
            {t`Play anyway`}
          </Button>
        </DialogFooter>
      }
    >
      <Typography sx={{ color: 'text.secondary' }}>
        {t`${{ game: gameName }} is now ${{ installed: warn?.installed ?? '' }}. This profile last launched on ${{ recorded: warn?.recorded ?? '' }}.`}
      </Typography>
      {(warn?.broken.length ?? 0) > 0 ? (
        <List dense={true} sx={{ mt: space.gap }}>
          {warn?.broken.map((mod) => (
            <ListItem key={modId(mod)} disableGutters={true}>
              <ListItemText
                primary={mod.name}
                secondary={
                  mod.brokeIn ? t`Broke in ${{ brokeIn: mod.brokeIn }}` : t`Broken for this version`
                }
              />
            </ListItem>
          ))}
        </List>
      ) : (
        <Typography sx={{ mt: space.gap, color: 'text.secondary' }}>
          {t`None of this profile's mods are marked broken for the new version.`}
        </Typography>
      )}
    </AppDialog>
  )
}
