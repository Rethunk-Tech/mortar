import { Box, CircularProgress, Typography } from '@mui/material'
import type { ReactNode } from 'react'

export function LoadingRow({ children }: { children: ReactNode }) {
  return (
    <Box
      role="status"
      sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 1, p: 2 }}
    >
      <CircularProgress size={16} aria-hidden={true} />
      <Typography sx={{ color: 'text.secondary' }}>{children}</Typography>
    </Box>
  )
}
