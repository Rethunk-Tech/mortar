import { useLingui } from '@lingui/react/macro'
import { Checkbox, DialogContentText, FormControlLabel } from '@mui/material'
import { ArchiveRestore, HardDriveDownload } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  BackupDialog,
  BackupSize,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { restoreBackup } from '../profiles/restoreBackup.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { openProfileDialog } from './profileDialog.ts'

export function BackupMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const hasGame = useProfiles((s) => s.game !== null)
  if (!hasGame) {
    return null
  }
  return (
    <MenuAction
      icon={<HardDriveDownload size={16} />}
      label={t`Back up to a file…`}
      onClick={() => {
        close()
        openProfileDialog('backup', profile.id)
      }}
    />
  )
}

// Restoring makes a new profile from the file, so it sits beside Back up rather than acting on this profile.
export function RestoreMenuItem({ close }: { close: () => void }) {
  const { t } = useLingui()
  const hasGame = useProfiles((s) => s.game !== null)
  if (!hasGame) {
    return null
  }
  return (
    <MenuAction
      icon={<ArchiveRestore size={16} />}
      label={t`Restore from a file…`}
      onClick={() => {
        close()
        restoreBackup().catch(reportUnexpected)
      }}
    />
  )
}

export function BackupProfileDialog({
  profile,
  open,
  onClose,
}: {
  profile: Profile
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const [mods, setMods] = useState(true)
  const [size, setSize] = useState<number | null>(null)
  useEffect(() => {
    if (!(open && game)) {
      return
    }
    let stale = false
    setSize(null)
    BackupSize(game.id, profile.id, mods)
      .then((n) => !stale && setSize(n))
      .catch(reportError(t`Could not back up the profile`))
    return () => {
      stale = true
    }
  }, [open, game, profile.id, mods, t])
  if (!game) {
    return null
  }
  const backup = async () => {
    onClose()
    const path = await BackupDialog(game.id, profile.id, mods)
    if (path) {
      useToasts.getState().push({ kind: 'success', title: t`Profile backed up`, body: path })
    }
  }
  return (
    <ConfirmDialog
      open={open}
      title={t`Back up ${profile.name}?`}
      body={t`The backup holds up to ${size === null ? '…' : formatBytes(size)}: the profile, its settings and config files, and its own saves when it keeps them.`}
      confirmLabel={t`Choose where to save…`}
      confirmDisabled={size === null}
      onCancel={onClose}
      onConfirm={() => {
        backup().catch(reportError(t`Could not back up the profile`))
      }}
    >
      <FormControlLabel
        control={<Checkbox checked={mods} onChange={(_, on) => setMods(on)} />}
        label={t`Include mod files (larger, works offline)`}
      />
      {mods ? null : (
        <DialogContentText>
          {t`Mods a site can download again are left out; restoring downloads them.`}
        </DialogContentText>
      )}
    </ConfirmDialog>
  )
}
