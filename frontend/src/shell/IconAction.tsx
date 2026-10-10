import { IconButton, Tooltip } from '@mui/material'
import type { MouseEvent, ReactNode } from 'react'
import { DisabledReason } from './DisabledReason.tsx'

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
  const button = (
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
  )
  if (disabled && disabledTitle !== undefined) {
    return (
      <DisabledReason title={disabledTitle} disabled={true}>
        {button}
      </DisabledReason>
    )
  }
  return (
    <Tooltip title={label}>
      {/* A disabled button fires no pointer events, so the tooltip needs a live wrapper. */}
      <span>{button}</span>
    </Tooltip>
  )
}
