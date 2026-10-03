import { IconButton, Tooltip } from '@mui/material'
import type { MouseEvent, ReactNode } from 'react'

const BORDER = 'var(--mortar-hairline-22)'

export function IconAction({
  label,
  icon,
  onClick,
  disabled = false,
  disabledTitle,
  pressed,
  menu = false,
}: {
  label: string
  icon: ReactNode
  onClick: (e: MouseEvent<HTMLElement>) => void
  disabled?: boolean
  disabledTitle?: string
  pressed?: boolean
  menu?: boolean
}) {
  const on = pressed === true
  return (
    <Tooltip title={disabled && disabledTitle !== undefined ? disabledTitle : label}>
      {/* A disabled button fires no pointer events, so the tooltip needs a live wrapper. */}
      <span>
        <IconButton
          aria-label={label}
          aria-pressed={pressed}
          aria-haspopup={menu ? 'menu' : undefined}
          disabled={disabled}
          onClick={onClick}
          sx={{
            width: 32,
            height: 32,
            borderRadius: '6px',
            border: `1px solid ${on ? 'currentColor' : BORDER}`,
            color: on ? 'primary.main' : 'var(--mortar-ink)',
          }}
        >
          {icon}
        </IconButton>
      </span>
    </Tooltip>
  )
}
