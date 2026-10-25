import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { Palette, Pencil } from 'lucide-react'
import { useMemo } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuHeading } from '../shell/TitleMenu.tsx'
import { CoverMenuItems, MoreMenuItems, ProfileMenuItem } from './ProfileMenuItems.tsx'
import { openProfileDialog } from './profileDialog.ts'
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
  // Built once per profile: the menu re-renders as it opens and closes, and the same items let React skip them.
  const items = useMemo(
    () => [
      <MenuHeading key="heading">{t`This profile`}</MenuHeading>,
      <ProfileMenuItem
        key="rename"
        icon={<Pencil size={16} />}
        label={t`Rename`}
        onClick={() => {
          onClose()
          useProfiles.getState().open(profile.id)
          useRenameRequest.getState().request(profile.id)
        }}
      />,
      <ProfileMenuItem
        key="edit"
        icon={<Palette size={16} />}
        label={t`Edit profile…`}
        onClick={() => {
          onClose()
          openProfileDialog('edit', profile.id)
        }}
      />,
      <CoverMenuItems key="cover" game={game} profile={profile} close={onClose} />,
      <MoreMenuItems key="more" profile={profile} close={onClose} />,
    ],
    [game, profile, onClose, t],
  )
  return (
    <Menu
      open={position !== null}
      onClose={onClose}
      anchorReference="anchorPosition"
      anchorPosition={position ?? undefined}
    >
      {items}
    </Menu>
  )
}
