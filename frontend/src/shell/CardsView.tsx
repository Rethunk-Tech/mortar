import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import { space } from '../theme/density.ts'

// A tab's cards under its ControlsRow: a grid of cards, or one row each in list view. `fill` centres an empty state.
export function CardsView({
  list,
  fill = false,
  children,
}: {
  list: boolean
  fill?: boolean
  children: ReactNode
}) {
  return (
    <Box
      sx={{
        flex: fill ? 1 : undefined,
        display: list || fill ? 'flex' : 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))',
        flexDirection: 'column',
        gap: space.gap,
        px: space.gutter,
        pb: space.gutter,
      }}
    >
      {children}
    </Box>
  )
}
