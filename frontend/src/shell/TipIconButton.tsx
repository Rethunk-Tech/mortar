import { Box, IconButton } from '@mui/material'
import type { IconButtonProps } from '@mui/material/IconButton'
import type { MouseEvent, ReactNode } from 'react'
import { Tooltip } from './Tooltip.tsx'

/**
 * An icon button named by label, which its tooltip shows unless tip gives the tooltip more to say. fill makes the
 * button fill its slot, as the rail's full-width Play and Stop do.
 */
export function TipIconButton({
  label,
  tip,
  children,
  disabled = false,
  size = 'small',
  fill = false,
  onClick,
  ...rest
}: {
  label: string
  tip?: ReactNode
  children: ReactNode
  disabled?: boolean
  size?: IconButtonProps['size']
  fill?: boolean
  onClick: (e: MouseEvent<HTMLButtonElement>) => void
} & Omit<IconButtonProps, 'aria-label' | 'children' | 'disabled' | 'onClick' | 'size'>) {
  const button = (
    <IconButton aria-label={label} size={size} disabled={disabled} onClick={onClick} {...rest}>
      {children}
    </IconButton>
  )
  // A disabled button fires no pointer events, so the tooltip needs a live wrapper.
  return (
    <Tooltip title={tip ?? label}>
      {fill ? (
        <Box component="span" sx={{ display: 'flex' }}>
          {button}
        </Box>
      ) : (
        <span>{button}</span>
      )}
    </Tooltip>
  )
}
