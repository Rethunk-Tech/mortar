import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'

const COMPACT_GAP = 1
const REGULAR_GAP = 2
const COMPACT_PADDING = 2
const REGULAR_PADDING = 4
const COMPACT_ICON = 28
const REGULAR_ICON = 40
const COMPACT_TITLE = 17
const REGULAR_TITLE = 22

// EmptyState fills a tab that has nothing to show yet with what it is for and how to fill it.
export function EmptyState({
  icon,
  title,
  children,
  action,
  compact = false,
}: {
  icon: ReactNode
  title: string
  children: ReactNode
  action?: ReactNode
  compact?: boolean
}) {
  return (
    <Box
      sx={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: compact ? COMPACT_GAP : REGULAR_GAP,
        px: 3,
        py: compact ? COMPACT_PADDING : REGULAR_PADDING,
        textAlign: 'center',
        color: 'var(--mortar-ink-dim-60)',
      }}
    >
      <Box
        sx={{
          '& svg': {
            width: compact ? COMPACT_ICON : REGULAR_ICON,
            height: compact ? COMPACT_ICON : REGULAR_ICON,
          },
        }}
        aria-hidden={true}
      >
        {icon}
      </Box>
      <Typography
        sx={{
          fontSize: compact ? COMPACT_TITLE : REGULAR_TITLE,
          fontWeight: 700,
          color: 'text.primary',
        }}
      >
        {title}
      </Typography>
      <Typography sx={{ maxWidth: 480, fontSize: 15, lineHeight: 1.5, color: 'text.secondary' }}>
        {children}
      </Typography>
      {action}
    </Box>
  )
}
