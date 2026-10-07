import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import MenuItem from '@mui/material/MenuItem'
import type { ReactNode } from 'react'
import { OneTip } from './OneTip.tsx'

// The one destructive menu item look: error.main alone is too dark on the dark menu paper and reads as disabled.
const destructiveSx = {
  color: 'error.light',
  '&:hover': { bgcolor: 'rgba(244, 67, 54, 0.14)' },
} as const

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
