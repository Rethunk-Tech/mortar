import { useLingui } from '@lingui/react/macro'
import { ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Eye, EyeOff } from 'lucide-react'
import { SetGameHidden } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { reportError } from '../toasts/report.ts'
import type { GameMenuAt } from './gameMenuAt.ts'

export function GameRowMenu({ at, onClose }: { at: GameMenuAt | null; onClose: () => void }) {
  const { t } = useLingui()
  const hidden = useSettings((s) => (at ? (s.hiddenGames?.includes(at.id) ?? false) : false))
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
