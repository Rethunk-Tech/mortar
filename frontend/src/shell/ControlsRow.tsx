import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import { space } from '../theme/density.ts'

// The row of controls under a tab's header. Put a SearchField with `grow` in it so the filter takes the free width.
export function ControlsRow({ children }: { children: ReactNode }) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        px: space.gutter,
        py: space.gap,
        flexShrink: 0,
      }}
    >
      {children}
    </Box>
  )
}
