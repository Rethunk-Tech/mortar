import Tooltip from '@mui/material/Tooltip'
import type { ReactNode } from 'react'

/** Tooltip around a span so a disabled control can still show why. */
export function DisabledReason({
  title,
  disabled,
  children,
}: {
  title: string
  disabled: boolean
  children: ReactNode
}) {
  if (!disabled) {
    return children
  }
  return (
    <Tooltip title={title}>
      <span>{children}</span>
    </Tooltip>
  )
}
