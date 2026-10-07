import { useLingui } from '@lingui/react/macro'
import { Button, ButtonGroup, type ButtonGroupProps, Divider, Menu } from '@mui/material'
import { ChevronDown, Download, FolderOpen, PackagePlus } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { bundleApplied } from '../bundles/applied.ts'
import { ApplyBundleDialog } from '../bundles/dialogs.tsx'
import { MenuHeading } from '../game/MenuHeading.tsx'
import { ApplyTemplateMenuItem } from '../game/TemplateMenuItems.tsx'
import { openDownloadsDialog } from '../install/downloadsDialog.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { ExtraFolderDialog } from './ExtraFolderDialog.tsx'

// The chevron of the Add split button: archives from the downloads folder or the game's extra mods folder (when
// one is set), and whole sets of mods (a template, a bundle). The archive pick is the button beside it, so the menu
// does not repeat it.
export function ExtraFolderMenu({
  folder,
  blocked,
  blockedReason,
  children,
  ...group
}: {
  folder: string
  blocked: boolean
  blockedReason: string
  children: ReactNode
} & Pick<ButtonGroupProps, 'variant' | 'size'>) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const profile = useProfiles(openProfileOf)
  const [open, setOpen] = useState(false)
  const [bundleOpen, setBundleOpen] = useState(false)
  const close = () => setAnchor(null)
  return (
    <>
      <ButtonGroup {...group}>
        {children}
        <DisabledReason title={blockedReason} disabled={blocked}>
          <Button
            disabled={blocked}
            aria-label={t`More ways to add mods`}
            aria-haspopup="menu"
            aria-expanded={anchor !== null}
            onClick={(e) => setAnchor(e.currentTarget)}
            sx={{ minWidth: 30, px: 0.5 }}
          >
            <ChevronDown size={14} aria-hidden={true} />
          </Button>
        </DisabledReason>
      </ButtonGroup>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close} keepMounted={true}>
        <MenuHeading>{t`Add from`}</MenuHeading>
        <MenuAction
          icon={<Download size={16} aria-hidden={true} />}
          label={t`The downloads folder…`}
          onClick={() => {
            setAnchor(null)
            openDownloadsDialog()
          }}
        />
        {folder === '' ? null : (
          <MenuAction
            icon={<FolderOpen size={16} aria-hidden={true} />}
            label={t`The extra mods folder…`}
            onClick={() => {
              setAnchor(null)
              setOpen(true)
            }}
          />
        )}
        {profile ? <Divider /> : null}
        {profile ? <MenuHeading>{t`Add a set`}</MenuHeading> : null}
        {profile ? <ApplyTemplateMenuItem profile={profile} close={close} /> : null}
        {profile ? (
          <MenuAction
            icon={<PackagePlus size={16} aria-hidden={true} />}
            label={t`Add a bundle…`}
            onClick={() => {
              close()
              setBundleOpen(true)
            }}
          />
        ) : null}
      </Menu>
      {profile ? (
        <ApplyBundleDialog
          open={bundleOpen}
          game={game}
          profileName={profile.name}
          profileId={profile.id}
          onClose={() => setBundleOpen(false)}
          onApplied={(result) => bundleApplied(result, profile.id)}
        />
      ) : null}
      <ExtraFolderDialog open={open} game={game} folder={folder} onClose={() => setOpen(false)} />
    </>
  )
}
