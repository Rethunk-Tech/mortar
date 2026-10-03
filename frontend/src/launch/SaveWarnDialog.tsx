import { plural } from '@lingui/core/macro'
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
import { useProfiles } from '../profiles/store.ts'
import { addRecordedMods } from '../saves/recordedActions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useLaunch } from './store.ts'

export function SaveWarnDialog() {
  const { t } = useLingui()
  const warn = useLaunch((s) => s.saveWarn)
  const cancel = useLaunch((s) => s.dismissSaveWarn)
  const playAnyway = useLaunch((s) => s.playAnyway)
  const save = warn?.save
  const recorded = (save?.lastMissing ?? []).length > 0
  const missing = recorded ? (save?.lastMissing ?? []) : (save?.missing ?? [])
  const names = missing.map((m) => m.name || m.uniqueId)
  const listed = names.join(', ')
  const switchTo =
    recorded &&
    save?.lastProfileExists &&
    save.lastProfileId &&
    save.lastProfileId !== warn?.profile
      ? save.lastProfileId
      : ''
  return (
    <Dialog
      open={warn !== null}
      onClose={cancel}
      transitionDuration={0}
      slotProps={{ paper: { sx: { maxWidth: 480 } } }}
    >
      <DialogTitle>
        {recorded ? t`This save needs other mods` : t`Your last save needs other mods`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>
          {recorded
            ? t`This save was last played with ${plural(missing.length, { one: '# mod this profile lacks', other: '# mods this profile lacks' })} (${listed}).`
            : t`${save?.farmer ?? ''}'s farm (${save?.folder ?? ''}) was last played with mods this profile does not have on. Loading it without them can lose their items or break the save.`}
        </DialogContentText>
        {recorded ? null : (
          <List dense={true} sx={{ mt: 1 }}>
            {missing.map((m) => (
              <ListItem key={m.uniqueId} disableGutters={true}>
                <ListItemText
                  primary={
                    <span title={(m.name ?? '').trim() === '' ? m.uniqueId : undefined}>
                      {(m.name ?? '').trim() === '' ? t`Unknown mod` : m.name}
                    </span>
                  }
                  secondary={m.disabled ? t`Switched off in this profile` : t`Not in this profile`}
                />
              </ListItem>
            ))}
          </List>
        )}
      </DialogContent>
      <DialogActions sx={{ flexWrap: 'wrap', gap: 1 }}>
        <Button onClick={cancel} sx={{ whiteSpace: 'nowrap' }}>{t`Cancel`}</Button>
        {switchTo ? (
          <Button
            onClick={() => {
              useProfiles.getState().open(switchTo)
              cancel()
            }}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Switch profile`}
          </Button>
        ) : null}
        {recorded && warn && save ? (
          <Button
            onClick={() => {
              addRecordedMods(warn.game, warn.profile, save).catch(reportUnexpected)
            }}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Add them`}
          </Button>
        ) : (
          <Button onClick={useLaunch.getState().openSaves} sx={{ whiteSpace: 'nowrap' }}>
            {t`Open saves`}
          </Button>
        )}
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
