import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import {
  ChevronDown,
  Copy,
  FolderOpen,
  GitCompare,
  GitMerge,
  History,
  Palette,
  Trash2,
  UserCog,
} from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuHeading, MenuRule } from '../shell/TitleMenu.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { BackupMenuItem, RestoreMenuItem } from './BackupMenuItems.tsx'
import { HomeButton } from './HomeButton.tsx'
import { openProfileDialog } from './profileDialog.ts'
import { SaveTemplateMenuItem } from './TemplateMenuItems.tsx'

// The Profile menu on Home. The sidebar's profile menu keeps the rest (cover, check, export, send, saved groups).
export function ProfileMenu({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const duplicate = useProfiles((s) => s.duplicate)
  const openFolder = useProfiles((s) => s.openFolder)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const close = () => setAnchor(null)
  return (
    <>
      <HomeButton
        icon={<UserCog size={16} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        {t`Profile`}
        <ChevronDown size={14} aria-hidden={true} />
      </HomeButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        <MenuHeading>{t`This profile`}</MenuHeading>
        <MenuAction
          icon={<Palette size={16} />}
          label={t`Edit profile…`}
          onClick={() => {
            close()
            openProfileDialog('edit', profile.id)
          }}
        />
        <MenuAction
          icon={<FolderOpen size={16} />}
          label={t`Open profile folder`}
          onClick={() => {
            close()
            openFolder(profile.id).catch(reportUnexpected)
          }}
        />
        <MenuRule />
        <MenuHeading>{t`Copies`}</MenuHeading>
        <MenuAction
          icon={<Copy size={16} />}
          label={t`Duplicate`}
          onClick={() => {
            close()
            duplicate(profile.id).catch(reportUnexpected)
          }}
        />
        <MenuAction
          icon={<GitMerge size={16} />}
          label={t`Add these mods to another profile…`}
          disabled={profiles.length < 2}
          reason={profiles.length < 2 ? t`No other profile to add them to.` : undefined}
          onClick={() => {
            close()
            openProfileDialog('merge', profile.id)
          }}
        />
        <SaveTemplateMenuItem profile={profile} close={close} />
        <BackupMenuItem profile={profile} close={close} />
        <RestoreMenuItem close={close} />
        <MenuRule />
        <MenuHeading>{t`Review`}</MenuHeading>
        <MenuAction
          icon={<History size={16} />}
          label={t`History…`}
          onClick={() => {
            close()
            openProfileDialog('history', profile.id)
          }}
        />
        <MenuAction
          icon={<GitCompare size={16} />}
          label={t`Compare…`}
          onClick={() => {
            close()
            openProfileDialog('compare', profile.id)
          }}
        />
        <MenuRule />
        <MenuAction
          icon={<Trash2 size={16} />}
          label={t`Delete profile…`}
          tone="error"
          onClick={() => {
            close()
            openProfileDialog('delete', profile.id)
          }}
        />
      </Menu>
    </>
  )
}
