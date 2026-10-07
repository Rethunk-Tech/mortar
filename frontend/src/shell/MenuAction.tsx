import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import MenuItem from '@mui/material/MenuItem'
import type { ReactNode } from 'react'
import { destructiveSx } from './destructive.ts'
import { OneTip } from './OneTip.tsx'

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
    <MenuItem disabled={disabled} onClick={onClick} sx={tone ? destructiveSx : undefined}>
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
    <OneTip title={tooltip} placement="left" describeChild={true}>
      <span>{item}</span>
    </OneTip>
  ) : (
    item
  )
}
