import { useLingui } from '@lingui/react/macro'
import { Button, ButtonGroup, type ButtonGroupProps, Menu } from '@mui/material'
import { ChevronDown, Download, PackagePlus } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { ApplyTemplateMenuItem } from '../game/TemplateMenuItems.tsx'
import { savedGroupApplied } from '../groups/applied.ts'
import { ApplySavedGroupDialog } from '../groups/dialogs.tsx'
import { openDownloadsDialog } from '../install/downloadsDialog.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { MenuHeading, MenuRule } from '../shell/TitleMenu.tsx'

// The chevron of the Add split button: archives from the downloads folder, and whole sets of mods (a template, a saved
// group). The archive pick is the button beside it, so the menu does not repeat it.
export function AddMenu({
  blocked,
  blockedReason,
  children,
  ...group
}: {
  blocked: boolean
  blockedReason: string
  children: ReactNode
} & Pick<ButtonGroupProps, 'variant' | 'size'>) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  // The id and name, not the profile, so a save of the open profile does not re-render the Add button.
  const profileId = useProfiles((s) => openProfileOf(s)?.id ?? '')
  const profileName = useProfiles((s) => openProfileOf(s)?.name ?? '')
  const [savedGroupOpen, setSavedGroupOpen] = useState(false)
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
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        <MenuHeading>{t`Add from`}</MenuHeading>
        <MenuAction
          icon={<Download size={16} aria-hidden={true} />}
          label={t`The downloads folder…`}
          onClick={() => {
            setAnchor(null)
            openDownloadsDialog()
          }}
        />
        {profileId ? <MenuRule /> : null}
        {profileId ? <MenuHeading>{t`Add a set`}</MenuHeading> : null}
        {profileId ? <ApplyTemplateMenuItem profile={{ id: profileId }} close={close} /> : null}
        {profileId ? (
          <MenuAction
            icon={<PackagePlus size={16} aria-hidden={true} />}
            label={t`Add a saved group…`}
            onClick={() => {
              close()
              setSavedGroupOpen(true)
            }}
          />
        ) : null}
      </Menu>
      {profileId ? (
        <ApplySavedGroupDialog
          open={savedGroupOpen}
          game={game}
          profileName={profileName}
          profileId={profileId}
          onClose={() => setSavedGroupOpen(false)}
          onApplied={(result) => savedGroupApplied(result, profileId)}
        />
      ) : null}
    </>
  )
}
