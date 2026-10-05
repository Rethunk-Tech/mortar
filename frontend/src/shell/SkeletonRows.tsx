import type { SxProps, Theme } from '@mui/material'
import { Box, Skeleton } from '@mui/material'

const GAP = 0.75

/** Placeholder rows of a fixed height, announced as one loading status. Pass `sx` for a grid layout. */
export function SkeletonRows({
  label,
  count,
  height,
  sx,
}: {
  label: string
  count: number
  height: number
  sx?: SxProps<Theme>
}) {
  return (
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
}
