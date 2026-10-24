import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  CheckFolder,
  MoveHome,
  MoveToFolder,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { useProfiles } from './store.ts'

/** Where a profile lives: under Mortar's data folder, or in a folder of its own (another drive), with move and check. */
export function ProfileFolderDialog({
  profile,
  game,
  open,
  onClose,
}: {
  profile: Profile
  game: string
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const [busy, runPending] = usePending()
  const folder = profile.folder ?? ''
  const { data: check } = useLoaded(
    open && folder !== '' ? () => CheckFolder(game, profile.id) : null,
    [open, folder, profile.id, game],
    null,
  )
  const run = (work: () => Promise<Profile>) =>
    runPending(() => work().then((next) => useProfiles.getState().replace(next)), {
      errorTitle: t`Could not move the profile`,
    })
  const openFolder = useProfiles((s) => s.openFolder)
  const pick = () =>
    PickFolder(t`Choose where ${{ name: profile.name }} lives`)
      .then((parent) => parent && run(() => MoveToFolder(game, profile.id, parent)))
      .catch(reportError(t`Could not open the file dialog`))
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Folder of ${{ name: profile.name }}`}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
          <Typography sx={{ overflowWrap: 'anywhere' }}>
            {folder ? t`Kept in ${{ path: folder }}.` : t`Kept in Mortar's data folder.`}
          </Typography>
          {check && !check.ok ? <Alert severity="error">{check.problem}</Alert> : null}
          <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
            {t`Moving copies the whole profile, then removes the old copy, so it needs room for both. On another drive the mods are copies, not links to the shared store, so the profile takes its full size there. Stop the game first.`}
          </Typography>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Close`}
        </Button>
        <Button
          disabled={busy}
          onClick={() => {
            openFolder(profile.id).catch(reportUnexpected)
          }}
        >
          {t`Open folder`}
        </Button>
        {folder ? (
          <Button disabled={busy} onClick={() => run(() => MoveHome(game, profile.id))}>
            {t`Move back to the data folder`}
          </Button>
        ) : null}
        <Button variant="contained" disabled={busy} onClick={pick}>
          {busy ? t`Moving…` : t`Move to a folder…`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
