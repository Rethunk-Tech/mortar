import Box from '@mui/material/Box'
import { type ReactNode, useId } from 'react'
import { Tooltip } from './Tooltip.tsx'

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
  const reasonId = useId()
  if (!disabled) {
    return children
  }
  return (
    <Tooltip title={title} describeChild={true}>
      <Box component="span" role="group" tabIndex={0} aria-describedby={reasonId}>
        {children}
        <span id={reasonId} hidden={true}>
          {title}
        </span>
      </Box>
    </Tooltip>
  )
}
