import { Box } from '@mui/material'
import type { ReactNode } from 'react'

// The row of controls under a tab's header. Put a SearchField with `grow` in it so the filter takes the free width.
export function ControlsRow({ children }: { children: ReactNode }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, py: 1.25, flexShrink: 0 }}>
      {children}
    </Box>
  )
}
