import { useLingui } from '@lingui/react/macro'
import { ArchiveRestore, HardDriveDownload } from 'lucide-react'
import { useState } from 'react'
import {
  BackupDialog,
  BackupSize,
  RestoreDialog,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

function BackupMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const [size, setSize] = useState<number | null>(null)
  if (!game) {
    return null
  }
  const backup = async () => {
    setSize(null)
    const path = await BackupDialog(game.id, profile.id)
    if (path) {
      useToasts.getState().push({ kind: 'success', title: t`Profile backed up`, body: path })
    }
  }
  return (
    <>
      <MenuAction
        icon={<HardDriveDownload size={16} />}
        label={t`Back up to a file…`}
        onClick={() => {
          close()
          BackupSize(game.id, profile.id)
            .then(setSize)
            .catch(reportError(t`Could not back up the profile`))
        }}
      />
      <ConfirmDialog
        open={size !== null}
        title={t`Back up ${profile.name}?`}
        body={t`The backup holds up to ${formatBytes(size ?? 0)}: the profile, its settings and config files, mods no site can download again, and its own saves when it keeps them. Other mods download again when it is restored.`}
        confirmLabel={t`Choose where to save…`}
        onCancel={() => setSize(null)}
        onConfirm={() => {
          backup().catch(reportError(t`Could not back up the profile`))
        }}
      />
    </>
  )
}

// Backs the profile up to one file, with the files no site can download again, and makes a new profile from such a
// file, whose other mods download again.
export function BackupMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  if (!game) {
    return null
  }
  const restore = async () => {
    const r = await RestoreDialog(game.id)
    if (!r.profile) {
      return
    }
    await useProfiles.getState().refresh()
    const missing = r.unavailable ?? []
    let body = t`Every mod is in place.`
    if (missing.length > 0) {
      body = t`Install these again by hand: ${listNames(missing)}`
    } else if (r.queued > 0) {
      body = t`${r.queued} mods are downloading.`
    }
    useToasts.getState().push({
      kind: missing.length > 0 ? 'warning' : 'success',
      title: t`Restored as ${r.name}`,
      body,
    })
  }
  return [
    <BackupMenuItem key="backup" profile={profile} close={close} />,
    <MenuAction
      key="restore"
      icon={<ArchiveRestore size={16} />}
      label={t`Restore from a file…`}
      onClick={() => {
        close()
        restore().catch(reportError(t`Could not restore the backup`))
      }}
    />,
  ]
}
