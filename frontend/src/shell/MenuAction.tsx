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
  onClick,
}: {
  icon: ReactNode
  label: ReactNode
  disabled?: boolean | undefined
  tooltip?: string | undefined
  onClick: () => void
}) {
  const item = (
    <MenuItem disabled={disabled} onClick={onClick}>
      <ListItemIcon sx={{ color: 'inherit', '& .MuiSvgIcon-root': { fontSize: 16 } }}>
        {icon}
      </ListItemIcon>
      <ListItemText>{label}</ListItemText>
    </MenuItem>
  )
  return tooltip ? (
    <Tooltip title={tooltip} placement="left">
      <span>{item}</span>
    </Tooltip>
  ) : (
    item
  )
}
