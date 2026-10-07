import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { ChevronDown, Copy, GitCompare, History, Palette, Trash2, UserCog } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { CompareDialog } from '../profiles/CompareDialog.tsx'
import { EditProfileDialog } from '../profiles/EditProfileDialog.tsx'
import { HistoryDialog } from '../profiles/HistoryDialog.tsx'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { BackupMenuItem } from './BackupMenuItems.tsx'
import { HomeButton } from './HomeButton.tsx'
import { MenuHeading } from './MenuHeading.tsx'
import { DeleteProfileDialog } from './ProfileMenuItems.tsx'
import { SaveTemplateMenuItem } from './TemplateMenuItems.tsx'

// The Profile menu on Home. The sidebar's profile menu keeps the rest (cover, check, export, send, bundles).
export function ProfileMenu({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const duplicate = useProfiles((s) => s.duplicate)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [editing, setEditing] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [compare, setCompare] = useState<{ a: Profile; b: Profile | null } | null>(null)
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
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close} keepMounted={true}>
        <MenuHeading>{t`This profile`}</MenuHeading>
        <MenuAction
          icon={<Palette size={16} />}
          label={t`Edit profile…`}
          onClick={() => {
            close()
            setEditing(true)
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
        <SaveTemplateMenuItem profile={profile} close={close} />
        <BackupMenuItem profile={profile} close={close} />
        <MenuRule />
        <MenuHeading>{t`Review`}</MenuHeading>
        <MenuAction
          icon={<History size={16} />}
          label={t`History…`}
          onClick={() => {
            close()
            setHistoryOpen(true)
          }}
        />
        <MenuAction
          icon={<GitCompare size={16} />}
          label={t`Compare…`}
          onClick={() => {
            close()
            setCompare({ a: profile, b: profiles.find((p) => p.id !== profile.id) ?? null })
          }}
        />
        <MenuRule />
        <MenuAction
          icon={<Trash2 size={16} />}
          label={t`Delete profile…`}
          tone="error"
          onClick={() => {
            close()
            setDeleting(true)
          }}
        />
      </Menu>
      <EditProfileDialog profile={profile} open={editing} onClose={() => setEditing(false)} />
      <HistoryDialog
        profileId={profile.id}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
      />
      <CompareDialog
        a={compare?.a ?? null}
        b={compare?.b ?? null}
        onClose={() => setCompare(null)}
      />
      <DeleteProfileDialog profile={profile} open={deleting} onClose={() => setDeleting(false)} />
    </>
  )
}
