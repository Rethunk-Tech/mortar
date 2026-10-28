import { Button, type ButtonProps, Menu } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import type { ReactNode } from 'react'
import { Tooltip } from './Tooltip.tsx'

/** An outlined button that drops a menu down from itself. The caller holds the anchor so its items can close it. */
export function MenuButton({
  anchor,
  setAnchor,
  icon,
  label,
  ariaLabel,
  tooltip,
  color,
  sx,
  children,
}: {
  anchor: HTMLElement | null
  setAnchor: (anchor: HTMLElement | null) => void
  icon: ReactNode
  label: ReactNode
  ariaLabel?: string
  tooltip?: string
  color?: ButtonProps['color']
  sx?: ButtonProps['sx']
  children: ReactNode
}) {
  const button = (
    <Button
      variant="outlined"
      {...(color === undefined ? {} : { color })}
      {...(ariaLabel === undefined ? {} : { 'aria-label': ariaLabel })}
      {...(sx === undefined ? {} : { sx })}
      startIcon={icon}
      endIcon={<ChevronDown size={14} aria-hidden={true} />}
      aria-haspopup="menu"
      aria-expanded={anchor !== null}
      onClick={(e) => setAnchor(e.currentTarget)}
    >
      {label}
    </Button>
  )
  return (
    <>
      {tooltip === undefined ? button : <Tooltip title={tooltip}>{button}</Tooltip>}
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {children}
      </Menu>
    </>
  )
}
