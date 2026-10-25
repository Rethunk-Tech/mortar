import MenuItem from '@mui/material/MenuItem'
import type { ReactNode } from 'react'
import { destructiveSx } from './destructive.ts'
import { pickWithoutFocus } from './pickWithoutFocus.ts'

// The row's parts are plain spans styled from the row rather than ListItemIcon and ListItemText: a menu redraws every
// row as focus moves through it, and each of those is several components more to redraw.
const rowSx = {
  '& .menu-action-icon': { minWidth: 36, flexShrink: 0, display: 'inline-flex' },
  '& .menu-action-text': { flex: '1 1 auto', minWidth: 0 },
  '& .menu-action-reason': {
    display: 'block',
    whiteSpace: 'normal',
    typography: 'body2',
    color: 'text.secondary',
  },
} as const

export function MenuAction({
  icon,
  label,
  disabled,
  reason,
  tone,
  onClick,
}: {
  icon: ReactNode
  label: ReactNode
  disabled?: boolean | undefined
  // Why the item is disabled, shown under its label; a disabled item cannot be hovered reliably for a tooltip.
  reason?: string | undefined
  tone?: 'error' | undefined
  onClick: () => void
}) {
  return (
    <MenuItem
      disabled={disabled}
      onClick={onClick}
      onMouseDown={pickWithoutFocus}
      sx={tone ? [rowSx, destructiveSx] : rowSx}
    >
      <span className="menu-action-icon">{icon}</span>
      <span className="menu-action-text">
        {label}
        {disabled && reason ? <span className="menu-action-reason">{reason}</span> : null}
      </span>
    </MenuItem>
  )
}
