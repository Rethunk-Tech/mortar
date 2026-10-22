import { alpha, type Theme } from '@mui/material'

// The one destructive menu item look: error.main alone is too dark on the dark menu paper and reads as disabled.
const HOVER_ALPHA = 0.14
export const destructiveSx = {
  color: 'error.light',
  '&:hover': { bgcolor: (theme: Theme) => alpha(theme.palette.error.light, HOVER_ALPHA) },
} as const
