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
import { SetSkipPlayCheck } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { LastSaveGap } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { addRecordedMods, enableRecordedMods } from '../saves/recordedActions.ts'
import { space } from '../theme/density.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import type { PlayIssueGroup } from './playIssues.ts'
import { overflowIssueCount } from './playIssues.ts'
import { cancelling } from './playModeState.ts'
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
    case 'saveModsOff':
      return t`This save was last played with ${plural(group.count, { one: '# mod that is disabled here', other: '# mods that are disabled here' })}`
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
      {group.kind === 'saveMods' || group.kind === 'saveModsOff' ? (
        <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
          {extra > 0
            ? t`${listNames(group.names)} and ${{ count: extra }} more`
            : listNames(group.names)}
        </Typography>
      ) : null}
      {group.names.length === 0 ||
      group.kind === 'saveMods' ||
      group.kind === 'saveModsOff' ? null : (
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
              <ListItemText primary={t`and ${{ count: extra }} more`} />
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
  const quit = cancelling(cancel)
  const playAnyway = useLaunch((s) => s.playAnyway)
  const openProblems = useLaunch((s) => s.openProblems)
  const updateAndPlay = useLaunch((s) => s.updateAndPlay)
  const skip = check?.skipPlayCheck ?? false
  const hasUpdates = (check?.groups ?? []).some((g) => g.kind === 'updates')
  const lastProfile = (check?.groups ?? []).find((g) => g.kind === 'lastProfile')
  const saveMods = (check?.groups ?? []).find((g) => g.kind === 'saveMods')
  const saveModsOff = (check?.groups ?? []).find((g) => g.kind === 'saveModsOff')
  const switchName = useProfiles(
    (s) => s.profiles.find((p) => p.id === lastProfile?.switchProfileId)?.name ?? '',
  )
  const [adding, runAdding] = usePending()
  const persistThen = (fn: () => void) => {
    if (check) {
      persistSkip(check.game, check.profile, check.skipPlayCheck)
    }
    fn()
  }
  return (
    <Dialog
      open={check !== null}
      onClose={() => persistThen(quit)}
      slotProps={{ paper: { sx: { maxWidth: 480 } } }}
    >
      <DialogTitle>{t`Before you play`}</DialogTitle>
      <DialogContent>
        <DialogContentText>{t`Check these before you play.`}</DialogContentText>
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
      <DialogActions sx={{ flexWrap: 'wrap', gap: space.gap }}>
        <Button onClick={() => persistThen(quit)}>{t`Cancel`}</Button>
        <Button onClick={() => persistThen(openProblems)}>{t`Open Problems`}</Button>
        {lastProfile?.switchProfileId ? (
          <Button
            onClick={() =>
              persistThen(() => {
                useProfiles.getState().open(lastProfile.switchProfileId ?? '')
                cancel()
              })
            }
          >
            {switchName ? t`Switch to ${switchName}` : t`Switch profile`}
          </Button>
        ) : null}
        {saveModsOff && check ? (
          <Button
            disabled={adding}
            onClick={() =>
              persistThen(() =>
                runAdding(async () => {
                  const [save] = await LastSaveGap(check.game, check.profile)
                  const { enabled, changes } = await enableRecordedMods(
                    check.game,
                    check.profile,
                    save,
                  )
                  cancel()
                  useToasts.getState().push({
                    kind: 'success',
                    title: plural(enabled, { one: 'Enabled # mod', other: 'Enabled # mods' }),
                    body: i18n._(msg`Press Play to start.`),
                    changes,
                  })
                }),
              )
            }
          >
            {t`Enable them`}
          </Button>
        ) : null}
        {saveMods && check ? (
          <Button
            disabled={adding}
            onClick={() =>
              persistThen(() =>
                runAdding(async () => {
                  const [save] = await LastSaveGap(check.game, check.profile)
                  await addRecordedMods(check.game, check.profile, save, false)
                  cancel()
                  useQueue.getState().setOpen(true)
                  useToasts.getState().push({
                    kind: 'info',
                    title: i18n._(msg`Adding the missing mods`),
                    body: i18n._(msg`Press Play when the downloads finish.`),
                  })
                }),
              )
            }
          >
            {t`Add the missing mods`}
          </Button>
        ) : null}
        <Button
          variant={hasUpdates ? 'text' : 'contained'}
          onClick={() => persistThen(() => playAnyway().catch(reportUnexpected))}
        >
          {t`Play anyway`}
        </Button>
        {hasUpdates ? (
          <Button
            variant="contained"
            onClick={() => persistThen(() => updateAndPlay().catch(reportUnexpected))}
          >
            {t`Update and play`}
          </Button>
        ) : null}
      </DialogActions>
    </Dialog>
  )
}
