import { useLingui } from '@lingui/react/macro'
import { Divider, Menu } from '@mui/material'
import { Palette, Pencil } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { EditProfileDialog } from '../profiles/EditProfileDialog.tsx'
import { HistoryDialog } from '../profiles/HistoryDialog.tsx'
import { useProfiles } from '../profiles/store.ts'
import { CoverMenuItems, MoreMenuItems, ProfileMenuItem } from './ProfileMenuItems.tsx'
import { useRenameRequest } from './renameRequest.ts'

// ProfileContextMenu is a sidebar profile's right-click menu: the same actions as the profile page's buttons.
export function ProfileContextMenu({
  game,
  profile,
  position,
  onClose,
}: {
  game: string
  profile: Profile
  position: { top: number; left: number } | null
  onClose: () => void
}) {
  const { t } = useLingui()
  const [editing, setEditing] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  return (
    <>
      <Menu
        open={position !== null}
        onClose={onClose}
        anchorReference="anchorPosition"
        anchorPosition={position ?? undefined}
        transitionDuration={0}
      >
        <ProfileMenuItem
          icon={<Pencil size={16} />}
          label={t`Rename`}
          onClick={() => {
            onClose()
            useProfiles.getState().open(profile.id)
            useRenameRequest.getState().request(profile.id)
          }}
        />
        <ProfileMenuItem
          icon={<Palette size={16} />}
          label={t`Edit profile`}
          onClick={() => {
            onClose()
            setEditing(true)
          }}
        />
        <CoverMenuItems game={game} profile={profile} close={onClose} />
        <Divider />
        <MoreMenuItems profile={profile} close={onClose} onHistory={() => setHistoryOpen(true)} />
      </Menu>
      <EditProfileDialog profile={profile} open={editing} onClose={() => setEditing(false)} />
      <HistoryDialog
        profileId={profile.id}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
      />
    </>
  )
}
