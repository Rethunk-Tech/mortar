import type { SxProps, Theme } from '@mui/material'
import { Box, Skeleton } from '@mui/material'
import { AfterDelay } from './AfterDelay.tsx'

const GAP = 0.75

/**
 * Placeholder rows of a fixed height, announced as one loading status. Pass `sx` for a grid layout. A dialog passes
 * `immediate`: its body would otherwise open blank, where a tab already shows its page around the wait.
 */
export function SkeletonRows({
  label,
  count,
  height,
  sx,
  immediate = false,
}: {
  label: string
  count: number
  height: number
  sx?: SxProps<Theme>
  immediate?: boolean
}) {
  const rows = (
    <Box
      role="status"
      aria-label={label}
      sx={[
        { display: 'flex', flexDirection: 'column', gap: GAP },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {Array.from({ length: count }, (_, n) => `row-${n}`).map((key) => (
        <Skeleton key={key} variant="rounded" height={height} />
      ))}
    </Box>
  )
  return immediate ? rows : <AfterDelay>{rows}</AfterDelay>
}
