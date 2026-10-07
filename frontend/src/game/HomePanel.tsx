import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { compact } from './compact.ts'

// One card of the Home grid; the grid is three columns wide and one in the compact layout.
export function HomePanel({
  title,
  span = 1,
  children,
}: {
  title: string
  span?: 1 | 2
  children: ReactNode
}) {
  return (
    <Box
      component="section"
      aria-label={title}
      sx={{
        gridColumn: `span ${span}`,
        [compact]: { gridColumn: 'auto' },
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        gap: 1,
        p: 2,
        minWidth: 0,
        bgcolor: 'var(--mortar-panel-85)',
        border: '1px solid var(--mortar-hairline)',
        borderRadius: '10px',
      }}
    >
      <Typography
        component="h2"
        sx={{ fontSize: 12, fontWeight: 700, letterSpacing: '0.06em', color: 'text.secondary' }}
      >
        {title.toUpperCase()}
      </Typography>
      {children}
    </Box>
  )
}
