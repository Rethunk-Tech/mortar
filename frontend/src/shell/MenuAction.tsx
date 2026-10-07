import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import MenuItem from '@mui/material/MenuItem'
import Tooltip from '@mui/material/Tooltip'
import type { ReactNode } from 'react'

export function MenuAction({
  icon,
  label,
  disabled,
  tooltip,
  tone,
  onClick,
}: {
  icon: ReactNode
  label: ReactNode
  disabled?: boolean | undefined
  tooltip?: string | undefined
  tone?: 'error' | undefined
  onClick: () => void
}) {
  const item = (
    <MenuItem
      disabled={disabled}
      onClick={onClick}
      sx={tone ? { color: `${tone}.main` } : undefined}
    >
      <ListItemIcon sx={{ color: 'inherit' }}>{icon}</ListItemIcon>
      {disabled && tooltip ? (
        <ListItemText
          secondary={tooltip}
          slotProps={{ secondary: { sx: { whiteSpace: 'normal' } } }}
        >
          {label}
        </ListItemText>
      ) : (
        <ListItemText>{label}</ListItemText>
      )}
    </MenuItem>
  )
  // A disabled item cannot be focused or hovered reliably, so its reason is shown as text instead of a tooltip.
  return tooltip && !disabled ? (
    <Tooltip title={tooltip} placement="left" describeChild={true}>
      <span>{item}</span>
    </Tooltip>
  ) : (
    item
  )
}
