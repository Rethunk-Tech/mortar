import { Button, type ButtonProps, Menu, type SxProps, type Theme } from '@mui/material'
import type { usePopoverAnchor } from '@rethunk/react-load-hooks'
import { ChevronDown } from 'lucide-react'
import type { ReactNode } from 'react'
import { Tooltip } from './Tooltip.tsx'

/** An outlined button that drops a menu down from itself. The caller holds the anchor so its items can close it. */
export function MenuButton({
  menu: { anchor, show, hide },
  icon,
  label,
  ariaLabel,
  tooltip,
  color,
  variant = 'outlined',
  size,
  disabled,
  paperSx,
  sx,
  children,
}: {
  menu: ReturnType<typeof usePopoverAnchor>
  icon?: ReactNode
  label: ReactNode
  ariaLabel?: string
  tooltip?: string
  color?: ButtonProps['color']
  variant?: ButtonProps['variant']
  size?: ButtonProps['size']
  disabled?: boolean
  paperSx?: SxProps<Theme>
  sx?: ButtonProps['sx']
  children: ReactNode
}) {
  const button = (
    <Button
      variant={variant}
      {...(size === undefined ? {} : { size })}
      {...(disabled === undefined ? {} : { disabled })}
      {...(color === undefined ? {} : { color })}
      {...(ariaLabel === undefined ? {} : { 'aria-label': ariaLabel })}
      {...(sx === undefined ? {} : { sx })}
      {...(icon === undefined ? {} : { startIcon: icon })}
      endIcon={<ChevronDown size={14} aria-hidden={true} />}
      aria-haspopup="menu"
      aria-expanded={anchor !== null}
      onClick={show}
    >
      {label}
    </Button>
  )
  return (
    <>
      {tooltip === undefined ? button : <Tooltip title={tooltip}>{button}</Tooltip>}
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={hide}
        {...(paperSx === undefined ? {} : { slotProps: { paper: { sx: paperSx } } })}
      >
        {children}
      </Menu>
    </>
  )
}
