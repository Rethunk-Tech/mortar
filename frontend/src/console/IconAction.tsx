import { IconButton, Tooltip } from '@mui/material'
import type { ReactNode } from 'react'

const BORDER = 'rgba(255,255,255,0.22)'

export function IconAction({
  label,
  icon,
  onClick,
  disabled = false,
  pressed,
}: {
  label: string
  icon: ReactNode
  onClick: () => void
  disabled?: boolean
  pressed?: boolean
}) {
  const on = pressed === true
  return (
    <Tooltip title={label}>
      {/* A disabled button fires no pointer events, so the tooltip needs a live wrapper. */}
      <span>
        <IconButton
          aria-label={label}
          aria-pressed={pressed}
          disabled={disabled}
          onClick={onClick}
          sx={{
            width: 32,
            height: 32,
            borderRadius: '6px',
            border: `1px solid ${on ? 'currentColor' : BORDER}`,
            color: on ? 'primary.main' : '#ffffff',
          }}
        >
          {icon}
        </IconButton>
      </span>
    </Tooltip>
  )
}
