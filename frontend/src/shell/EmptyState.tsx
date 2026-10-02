import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'

// EmptyState fills a tab that has nothing to show yet with what it is for and how to fill it.
export function EmptyState({
  icon,
  title,
  children,
  action,
}: {
  icon: ReactNode
  title: string
  children: ReactNode
  action?: ReactNode
}) {
  return (
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 2,
        px: 3,
        py: 4,
        textAlign: 'center',
        color: 'rgba(255,255,255,0.6)',
      }}
    >
      {icon}
      <Typography sx={{ fontSize: 22, fontWeight: 700, color: 'text.primary' }}>{title}</Typography>
      <Typography sx={{ maxWidth: 480, fontSize: 15, lineHeight: 1.5, color: 'text.secondary' }}>
        {children}
      </Typography>
      {action}
    </Box>
  )
}
