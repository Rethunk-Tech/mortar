import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import { space } from '../theme/density.ts'

export function Panel({ children, width = 680 }: { children: ReactNode; width?: number }) {
  return (
    <Box
      sx={{
        width: `min(${width}px, calc(100% - 32px))`,
        display: 'flex',
        flexDirection: 'column',
        gap: space.pad,
        p: space.pad,
        bgcolor: 'var(--mortar-paper-78)',
        borderRadius: '8px',
        flexShrink: 0,
      }}
    >
      {children}
    </Box>
  )
}
