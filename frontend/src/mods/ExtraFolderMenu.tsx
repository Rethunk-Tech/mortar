import { useLingui } from '@lingui/react/macro'
import {
  Button,
  ButtonGroup,
  type ButtonGroupProps,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
} from '@mui/material'
import { ChevronDown, Download, FolderOpen, Plus } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { openDownloadsDialog } from '../install/downloadsDialog.ts'
import { useInstall } from '../install/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { ExtraFolderDialog } from './ExtraFolderDialog.tsx'

// The chevron of the Add split button: the same archive pick, an archive from the downloads folder, or the mods in
// the game's extra mods folder when one is set.
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
  const pick = useInstall((s) => s.pick)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [open, setOpen] = useState(false)
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
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
        <MenuItem
          onClick={() => {
            setAnchor(null)
            pick().catch(reportUnexpected)
          }}
        >
          <ListItemIcon>
            <Plus size={16} aria-hidden={true} />
          </ListItemIcon>
          <ListItemText>{t`Archive…`}</ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => {
            setAnchor(null)
            openDownloadsDialog()
          }}
        >
          <ListItemIcon>
            <Download size={16} aria-hidden={true} />
          </ListItemIcon>
          <ListItemText>{t`From the downloads folder…`}</ListItemText>
        </MenuItem>
        {folder === '' ? null : (
          <MenuItem
            onClick={() => {
              setAnchor(null)
              setOpen(true)
            }}
          >
            <ListItemIcon>
              <FolderOpen size={16} aria-hidden={true} />
            </ListItemIcon>
            <ListItemText>{t`From the extra mods folder…`}</ListItemText>
          </MenuItem>
        )}
      </Menu>
      <ExtraFolderDialog open={open} game={game} folder={folder} onClose={() => setOpen(false)} />
    </>
  )
}
