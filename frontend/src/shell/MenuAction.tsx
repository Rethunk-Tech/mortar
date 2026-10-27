import MenuItem from '@mui/material/MenuItem'
import { Check } from 'lucide-react'
import type { ReactNode } from 'react'
import { destructiveSx } from './destructive.ts'
import { pickWithoutFocus } from './pickWithoutFocus.ts'
import { Tooltip } from './Tooltip.tsx'

// The row's parts are plain spans styled from the row rather than ListItemIcon and ListItemText: a menu redraws every
// row as focus moves through it, and each of those is several components more to redraw.
const rowSx = {
  '& .menu-action-icon': { minWidth: 36, flexShrink: 0, display: 'inline-flex' },
  '& .menu-action-text': { flex: '1 1 auto', minWidth: 0 },
  '& .menu-action-end': { display: 'inline-flex', width: 16, ml: 2, flexShrink: 0 },
  '& .menu-action-detail': { display: 'block', typography: 'body2', color: 'text.secondary' },
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
  title,
  detail,
  end,
  expanded,
  onClick,
}: {
  icon: ReactNode
  label: ReactNode
  title?: string | undefined
  // A second line under the label, such as what the action does.
  detail?: ReactNode
  // Drawn at the row's end, such as the chevron of a row that opens more rows below it.
  end?: ReactNode
  // Set on a row that shows or hides the rows below it.
  expanded?: boolean | undefined
  disabled?: boolean | undefined
  // Why the item is disabled, shown under its label; a disabled item cannot be hovered reliably for a tooltip.
  reason?: string | undefined
  tone?: 'error' | undefined
  onClick: () => void
}) {
  return (
    <MenuItem
      disabled={disabled}
      title={title}
      aria-expanded={expanded}
      onClick={onClick}
      onMouseDown={pickWithoutFocus}
      sx={tone ? [rowSx, destructiveSx] : rowSx}
    >
      <span className="menu-action-icon">{icon}</span>
      <span className="menu-action-text">
        {label}
        {detail ? <span className="menu-action-detail">{detail}</span> : null}
        {disabled && reason ? <span className="menu-action-reason">{reason}</span> : null}
      </span>
      {end ? <span className="menu-action-end">{end}</span> : null}
    </MenuItem>
  )
}

/**
 * A menu row that is picked or not, a radio choice or a checkbox alike. The tick sits where an action's icon would, so
 * a menu mixing both lines up; a row with its own icon keeps it there and shows the tick at its end. Detail is a second
 * line under the label, such as a run's error; hint is a tooltip.
 */
export function MenuCheck({
  checked,
  label,
  icon,
  detail,
  hint,
  title,
  disabled,
  radio,
  onClick,
}: {
  checked: boolean
  label: ReactNode
  icon?: ReactNode
  detail?: ReactNode
  hint?: string | undefined
  title?: string | undefined
  disabled?: boolean | undefined
  // radio marks one choice of several, read out as such; otherwise the row is a checkbox.
  radio?: boolean
  onClick: () => void
}) {
  const tick = checked ? <Check size={16} aria-hidden={true} /> : null
  const row = (
    <MenuItem
      selected={checked}
      disabled={disabled}
      title={title}
      role={radio ? 'menuitemradio' : 'menuitemcheckbox'}
      aria-checked={checked}
      onClick={onClick}
      onMouseDown={pickWithoutFocus}
      sx={rowSx}
    >
      <span className="menu-action-icon">{icon ?? tick}</span>
      <span className="menu-action-text">
        {label}
        {detail ? <span className="menu-action-detail">{detail}</span> : null}
      </span>
      {icon ? <span className="menu-action-end">{tick}</span> : null}
    </MenuItem>
  )
  return hint ? (
    <Tooltip title={hint} placement="right">
      {row}
    </Tooltip>
  ) : (
    row
  )
}
