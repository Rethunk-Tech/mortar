import { Box } from '@mui/material'
import type { ReactNode } from 'react'

export function Panel({ children, width = 680 }: { children: ReactNode; width?: number }) {
  return (
    <Box
      sx={{
        width: `min(${width}px, calc(100% - 32px))`,
        display: 'flex',
        flexDirection: 'column',
        gap: '16px',
        p: '24px',
        bgcolor: 'rgba(50,50,60,0.78)',
        borderRadius: '8px',
        flexShrink: 0,
      }}
    >
      {children}
    </Box>
  )
}
