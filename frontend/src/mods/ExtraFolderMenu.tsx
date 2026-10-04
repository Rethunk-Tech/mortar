import { useLingui } from '@lingui/react/macro'
import { Button, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { ChevronDown, FolderOpen, Plus } from 'lucide-react'
import { useState } from 'react'
import { useInstall } from '../install/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ExtraFolderDialog } from './ExtraFolderDialog.tsx'

// The chevron of the Add split button: the same archive pick, or the mods in the game's extra mods folder.
export function ExtraFolderMenu({ folder, blocked }: { folder: string; blocked: boolean }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const pick = useInstall((s) => s.pick)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [open, setOpen] = useState(false)
  return (
    <>
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
            setOpen(true)
          }}
        >
          <ListItemIcon>
            <FolderOpen size={16} aria-hidden={true} />
          </ListItemIcon>
          <ListItemText>{t`From the extra mods folder…`}</ListItemText>
        </MenuItem>
      </Menu>
      <ExtraFolderDialog open={open} game={game} folder={folder} onClose={() => setOpen(false)} />
    </>
  )
}
