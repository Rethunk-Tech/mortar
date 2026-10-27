import { Dialog } from '@mui/material'
import type { KeyboardEvent, ReactNode } from 'react'
import { paperSx } from './frame.ts'

/**
 * A dialog whose content is its own frame, such as a screenshot viewer or a preview card: the dialogs' paper and
 * border, with no header or footer. It names itself through label or labelledBy, since it has no title of its own.
 */
export function BareDialog({
  open,
  onClose,
  label,
  labelledBy,
  width,
  onKeyDown,
  children,
}: {
  open: boolean
  onClose: () => void
  label?: string
  labelledBy?: string
  width?: number
  onKeyDown?: (e: KeyboardEvent) => void
  children: ReactNode
}) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth={false}
      aria-labelledby={labelledBy}
      onKeyDown={onKeyDown}
      slotProps={{
        paper: { 'aria-label': label, sx: [paperSx, width === undefined ? {} : { width }] },
      }}
    >
      {children}
    </Dialog>
  )
}
