import { msg } from '@lingui/core/macro'
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
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import type { PlayIssueGroup } from './playIssues.ts'
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
    default:
      return ''
  }
}

function persistSkip(game: string, profile: string, on: boolean) {
  SetSkipPlayCheck(game, profile, on)
    .then((next) => useProfiles.getState().replace(next))
    .catch((e: unknown) => {
      useToasts.getState().push({
        kind: 'error',
        title: i18n._(msg`Could not save the profile`),
        body: errorMessage(e),
      })
    })
}

function Group({ group }: { group: PlayIssueGroup }) {
  return (
    <>
      <Typography sx={{ mt: 1.5, fontWeight: 600 }}>
        <GroupHeading group={group} />
      </Typography>
      {group.names.length === 0 ? null : (
        <List dense={true}>
          {group.names.map((name) => (
            <ListItem key={name} disableGutters={true}>
              <ListItemText primary={name} />
            </ListItem>
          ))}
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
  return (
    <Dialog
      open={check !== null}
      onClose={cancel}
      transitionDuration={0}
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
                  persistSkip(check.game, check.profile, on)
                }
              }}
            />
          }
          label={t`Don't check before Play`}
        />
      </DialogContent>
      <DialogActions sx={{ flexWrap: 'wrap', gap: 1 }}>
        <Button onClick={cancel} sx={{ whiteSpace: 'nowrap' }}>{t`Cancel`}</Button>
        <Button onClick={openProblems} sx={{ whiteSpace: 'nowrap' }}>{t`Open problems`}</Button>
        {hasUpdates ? (
          <Button
            onClick={() => updateAndPlay().catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Update and play`}
          </Button>
        ) : null}
        {lastProfile?.switchProfileId ? (
          <Button
            onClick={() => {
              useProfiles.getState().open(lastProfile.switchProfileId ?? '')
              cancel()
            }}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Switch profile`}
          </Button>
        ) : null}
        <Button
          variant="contained"
          onClick={() => playAnyway().catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Play anyway`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
