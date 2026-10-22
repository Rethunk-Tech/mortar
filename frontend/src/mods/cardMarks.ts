import { alpha, type Theme } from '@mui/material/styles'

// A selected card is tinted as well as outlined, so a selection reads at a glance across a full grid.
const MARKED_FILL = 0.28

/** A selected card is outlined and tinted; a local one (added from a file, so no site updates it) has a dashed edge. */
export const cardMarks = (marked: boolean, local: boolean) => ({
  outline: marked ? '1px solid' : 'none',
  outlineColor: 'primary.main',
  bgcolor: marked ? (theme: Theme) => alpha(theme.palette.primary.main, MARKED_FILL) : undefined,
  ...(local ? { border: '1px dashed', borderColor: 'info.main' } : {}),
})
