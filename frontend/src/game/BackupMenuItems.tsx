import { useLingui } from '@lingui/react/macro'
import { ArchiveRestore, HardDriveDownload } from 'lucide-react'
import {
  BackupDialog,
  RestoreDialog,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { listNames } from '../i18n/list.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// Backs the profile up to one file without its mod files, and makes a new profile from such a file, whose mods
// download again.
export function BackupMenuItems({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  if (!game) {
    return null
  }
  const backup = async () => {
    const path = await BackupDialog(game.id, profile.id)
    if (path) {
      useToasts.getState().push({ kind: 'success', title: t`Profile backed up`, body: path })
    }
  }
  const restore = async () => {
    const r = await RestoreDialog(game.id)
    if (!r.profile) {
      return
    }
    await useProfiles.getState().refresh()
    const missing = r.unavailable ?? []
    useToasts.getState().push({
      kind: missing.length > 0 ? 'warning' : 'success',
      title: t`Restored as ${r.name}`,
      body:
        missing.length > 0
          ? t`Install these again by hand: ${listNames(missing)}`
          : t`${r.queued} mods are downloading.`,
    })
  }
  return [
    <MenuAction
      key="backup"
      icon={<HardDriveDownload size={16} />}
      label={t`Back up to a file…`}
      onClick={() => {
        close()
        backup().catch(reportError(t`Could not back up the profile`))
      }}
    />,
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
