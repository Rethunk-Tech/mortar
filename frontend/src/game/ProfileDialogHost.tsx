import { useLingui } from '@lingui/react/macro'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { savedGroupApplied } from '../groups/applied.ts'
import { ApplySavedGroupDialog } from '../groups/dialogs.tsx'
import { CompareDialog } from '../profiles/CompareDialog.tsx'
import { EditProfileDialog } from '../profiles/EditProfileDialog.tsx'
import { HealthDialog } from '../profiles/HealthDialog.tsx'
import { HistoryDialog } from '../profiles/HistoryDialog.tsx'
import { MergeDialog } from '../profiles/MergeDialog.tsx'
import { ProfileFolderDialog } from '../profiles/ProfileFolderDialog.tsx'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { ApplyTemplateDialog } from '../templates/ApplyTemplateDialog.tsx'
import { ManageTemplatesDialog, SaveTemplateDialog } from '../templates/TemplateDialogs.tsx'
import { useTemplates } from '../templates/useTemplates.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { BackupProfileDialog } from './BackupMenuItems.tsx'
import { FarmDialog } from './FarmMenuItem.tsx'
import { closeProfileDialog, type ProfileDialogKind, useProfileDialog } from './profileDialog.ts'

interface DialogProps {
  profile: Profile
  game: string
  open: boolean
}

function DeleteProfileDialog({ profile, open }: DialogProps) {
  const { t } = useLingui()
  const remove = useProfiles((s) => s.remove)
  return (
    <ConfirmDialog
      open={open}
      title={t`Delete ${{ name: profile.name }}?`}
      body={t`The profile stays restorable for 30 days from Recently deleted.`}
      confirmLabel={t`Delete`}
      color="error"
      onCancel={closeProfileDialog}
      onConfirm={() => {
        closeProfileDialog()
        remove(profile.id).catch(reportUnexpected)
      }}
    />
  )
}

function CompareWithNext({ profile, open }: DialogProps) {
  const other = useProfiles((s) => s.profiles.find((p) => p.id !== profile.id) ?? null)
  return <CompareDialog a={open ? profile : null} b={other} onClose={closeProfileDialog} />
}

function ApplyTemplate({ profile, game, open }: DialogProps) {
  const { templates } = useTemplates(game, open)
  return (
    <ApplyTemplateDialog
      open={open}
      game={game}
      profileId={profile.id}
      templates={templates}
      onClose={closeProfileDialog}
    />
  )
}

function ManageTemplates({ game, open }: DialogProps) {
  const { templates, reload } = useTemplates(game, open)
  return (
    <ManageTemplatesDialog
      open={open}
      game={game}
      templates={templates}
      onChanged={reload}
      onClose={closeProfileDialog}
    />
  )
}

function ProfileDialog({ kind, ...props }: DialogProps & { kind: ProfileDialogKind }) {
  const { profile, game, open } = props
  const onClose = closeProfileDialog
  switch (kind) {
    case 'edit':
      return <EditProfileDialog profile={profile} open={open} onClose={onClose} />
    case 'history':
      return <HistoryDialog profileId={profile.id} open={open} onClose={onClose} />
    case 'health':
      return <HealthDialog profileId={profile.id} open={open} onClose={onClose} />
    case 'compare':
      return <CompareWithNext {...props} />
    case 'merge':
      return <MergeDialog source={open ? profile : null} onClose={onClose} />
    case 'delete':
      return <DeleteProfileDialog {...props} />
    case 'savedGroup':
      return (
        <ApplySavedGroupDialog
          open={open}
          game={game}
          profileName={profile.name}
          profileId={profile.id}
          onClose={onClose}
          onApplied={(result) => savedGroupApplied(result, profile.id)}
        />
      )
    case 'folder':
      return <ProfileFolderDialog profile={profile} game={game} open={open} onClose={onClose} />
    case 'farm':
      return <FarmDialog open={open} profile={profile} onClose={onClose} />
    case 'backup':
      return <BackupProfileDialog profile={profile} open={open} onClose={onClose} />
    case 'saveTemplate':
      return (
        <SaveTemplateDialog
          open={open}
          game={game}
          profileId={profile.id}
          profileName={profile.name}
          onClose={onClose}
        />
      )
    case 'applyTemplate':
      return <ApplyTemplate {...props} />
    default:
      return <ManageTemplates {...props} />
  }
}

// Mounted once above every profile menu; each opening mounts its dialog fresh and keeps it through the closing fade.
export function ProfileDialogHost() {
  const kind = useProfileDialog((s) => s.kind)
  const open = useProfileDialog((s) => s.open)
  const seq = useProfileDialog((s) => s.seq)
  const profileId = useProfileDialog((s) => s.profileId)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const game = useProfiles((s) => s.game?.id ?? '')
  if (kind === null || profile === undefined) {
    return null
  }
  return <ProfileDialog key={seq} kind={kind} profile={profile} game={game} open={open} />
}
