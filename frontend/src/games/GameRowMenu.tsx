import { useLingui } from '@lingui/react/macro'
import { ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Eye, EyeOff } from 'lucide-react'
import type { MouseEvent } from 'react'
import { SetGameHidden } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportError } from '../toasts/report.ts'

export interface GameMenuAt {
  id: string
  top: number
  left: number
}

// Right-click on a row, or the menu key while a row holds focus, opens that game's menu.
export function gameMenuAt(e: MouseEvent<HTMLElement>): GameMenuAt | null {
  const tile = (e.target as HTMLElement).closest<HTMLElement>('[data-tile]')
  const id = tile?.dataset.game
  if (!(tile && id)) {
    return null
  }
  if (e.clientX !== 0 || e.clientY !== 0) {
    return { id, top: e.clientY, left: e.clientX }
  }
  const box = (document.activeElement ?? tile).getBoundingClientRect()
  return { id, top: box.bottom, left: box.left }
}

export function GameRowMenu({ at, onClose }: { at: GameMenuAt | null; onClose: () => void }) {
  const { t } = useLingui()
  const hidden = useSettings((s) => (at ? (s.hiddenGames ?? []).includes(at.id) : false))
  const toggle = () => {
    if (at) {
      SetGameHidden(at.id, !hidden).catch(reportError(t`Could not save that setting`))
    }
    onClose()
  }
  return (
    <Menu
      open={at !== null}
      onClose={onClose}
      anchorReference="anchorPosition"
      anchorPosition={at ? { top: at.top, left: at.left } : undefined}
    >
      <MenuItem onClick={toggle}>
        <ListItemIcon>{hidden ? <Eye size={16} /> : <EyeOff size={16} />}</ListItemIcon>
        <ListItemText>{hidden ? t`Unhide` : t`Hide this game`}</ListItemText>
      </MenuItem>
    </Menu>
  )
}
