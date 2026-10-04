import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  List,
  ListItem,
  ListItemText,
  Typography,
} from '@mui/material'
import { SetSkipPlayCheck } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { LastSaveGap } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { addRecordedMods } from '../saves/recordedActions.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import type { PlayIssueGroup } from './playIssues.ts'
import { overflowIssueCount } from './playIssues.ts'
import { useLaunch } from './store.ts'

function GroupHeading({ group }: { group: PlayIssueGroup }) {
  const { t } = useLingui()
  switch (group.kind) {
    case 'missing':
      return t`Missing required dependencies (${group.count})`
    case 'conflicts':
      return t`Conflicts (${group.count})`
    case 'updates':
      return t`Pending updates (${group.count})`
    case 'broken':
      return t`Broken or obsolete (${group.count})`
    case 'lastProfile':
      return t`${group.save} was last played with ${group.profileName}`
    case 'changes':
      return t`Changed since last run (${group.count})`
    case 'saveMods':
      return t`This save was last played with ${plural(group.count, { one: '# mod this profile lacks', other: '# mods this profile lacks' })}`
    default:
      return ''
  }
}

function persistSkip(game: string, profile: string, on: boolean) {
  SetSkipPlayCheck(game, profile, on)
    .then((next) => useProfiles.getState().replace(next))
    .catch(reportError(i18n._(msg`Could not save the profile`)))
}

function Group({ group }: { group: PlayIssueGroup }) {
  const { t } = useLingui()
  const extra = overflowIssueCount(group)
  return (
    <>
      <Typography sx={{ mt: 1.5, fontWeight: 600 }}>
        <GroupHeading group={group} />
      </Typography>
      {group.names.length === 0 ? null : (
        <List dense={true}>
          {group.names.map((name, i) => (
            <ListItem key={name} disableGutters={true}>
              <ListItemText
                primary={name}
                slotProps={{ primary: { title: group.nameTitles?.[i] } }}
              />
            </ListItem>
          ))}
          {extra > 0 ? (
            <ListItem disableGutters={true}>
              <ListItemText primary={t`and ${extra} more`} />
            </ListItem>
          ) : null}
        </List>
      )}
    </>
  )
}

export function PrePlayDialog() {
  const { t } = useLingui()
  const check = useLaunch((s) => s.playCheck)
  const cancel = useLaunch((s) => s.dismissPlayCheck)
  const playAnyway = useLaunch((s) => s.playAnyway)
  const openProblems = useLaunch((s) => s.openProblems)
  const updateAndPlay = useLaunch((s) => s.updateAndPlay)
  const skip = check?.skipPlayCheck ?? false
  const hasUpdates = (check?.groups ?? []).some((g) => g.kind === 'updates')
  const lastProfile = (check?.groups ?? []).find((g) => g.kind === 'lastProfile')
  const saveMods = (check?.groups ?? []).find((g) => g.kind === 'saveMods')
  const persistThen = (fn: () => void) => {
    if (check) {
      persistSkip(check.game, check.profile, check.skipPlayCheck)
    }
    fn()
  }
  return (
    <Dialog
      open={check !== null}
      onClose={() => persistThen(cancel)}
      slotProps={{ paper: { sx: { maxWidth: 480 } } }}
    >
      <DialogTitle>{t`Before you play`}</DialogTitle>
      <DialogContent>
        <DialogContentText>{t`This profile has problems that can affect a launch.`}</DialogContentText>
        {(check?.groups ?? []).map((group) => (
          <Group key={group.kind} group={group} />
        ))}
        <FormControlLabel
          sx={{ mt: 1 }}
          control={
            <Checkbox
              checked={skip}
              onChange={(_, on) => {
                if (check) {
                  useLaunch.setState({ playCheck: { ...check, skipPlayCheck: on } })
                }
              }}
            />
          }
          label={t`Don't check before Play`}
        />
      </DialogContent>
      <DialogActions sx={{ flexWrap: 'wrap', gap: 1 }}>
        <Button onClick={() => persistThen(cancel)}>{t`Cancel`}</Button>
        <Button onClick={() => persistThen(openProblems)}>{t`Open problems…`}</Button>
        {hasUpdates ? (
          <Button onClick={() => persistThen(() => updateAndPlay().catch(reportUnexpected))}>
            {t`Update and play`}
          </Button>
        ) : null}
        {lastProfile?.switchProfileId ? (
          <Button
            onClick={() =>
              persistThen(() => {
                useProfiles.getState().open(lastProfile.switchProfileId ?? '')
                cancel()
              })
            }
          >
            {t`Switch profile`}
          </Button>
        ) : null}
        {saveMods && check ? (
          <Button
            onClick={() =>
              persistThen(() => {
                LastSaveGap(check.game, check.profile)
                  .then(([save]) => addRecordedMods(check.game, check.profile, save))
                  .catch(reportUnexpected)
              })
            }
          >
            {t`Add them`}
          </Button>
        ) : null}
        <Button
          variant="contained"
          onClick={() => persistThen(() => playAnyway().catch(reportUnexpected))}
        >
          {t`Play anyway`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
