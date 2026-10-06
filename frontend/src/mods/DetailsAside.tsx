import { Drawer, useMediaQuery } from '@mui/material'
import type { ReactNode } from 'react'
import { compactQuery } from '../game/compact.ts'
import { ResizableAside } from './ResizableAside.tsx'

// The one details panel shell, for a mod in the profile and a hit in Browse: an aside at the window's normal width,
// a right drawer below 960px. It takes room only while open, so the list gets the full width otherwise.
export function DetailsAside({
  open,
  label,
  onClose,
  children,
}: {
  open: boolean
  label: string
  onClose: () => void
  children: ReactNode
}) {
  const narrow = useMediaQuery(compactQuery)
  if (narrow) {
    return (
      <Drawer
        anchor="right"
        open={open}
        onClose={onClose}
        sx={{ top: 'var(--title-bar)' }}
        slotProps={{
          paper: {
            role: 'dialog',
            'aria-label': label,
            sx: {
              width: 320,
              top: 'var(--title-bar)',
              height: 'calc(100% - var(--title-bar))',
              bgcolor: 'var(--mortar-panel-92)',
            },
          },
        }}
      >
        {open ? children : null}
      </Drawer>
    )
  }
  if (!open) {
    return null
  }
  return <ResizableAside label={label}>{children}</ResizableAside>
}
