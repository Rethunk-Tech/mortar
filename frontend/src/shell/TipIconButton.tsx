import { IconButton, Tooltip } from '@mui/material'
import type { IconButtonProps } from '@mui/material/IconButton'
import type { MouseEvent, ReactNode } from 'react'

export function TipIconButton({
  label,
  children,
  disabled = false,
  onClick,
  ...rest
}: {
  label: string
  children: ReactNode
  disabled?: boolean
  onClick: (e: MouseEvent<HTMLButtonElement>) => void
} & Omit<IconButtonProps, 'aria-label' | 'children' | 'disabled' | 'onClick' | 'size'>) {
  return (
    <Tooltip title={label}>
      {/* A disabled button fires no pointer events, so the tooltip needs a live wrapper. */}
      <span>
        <IconButton aria-label={label} size="small" disabled={disabled} onClick={onClick} {...rest}>
          {children}
        </IconButton>
      </span>
    </Tooltip>
  )
}
