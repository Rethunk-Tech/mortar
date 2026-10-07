// The one destructive menu item look: error.main alone is too dark on the dark menu paper and reads as disabled.
export const destructiveSx = {
  color: 'error.light',
  '&:hover': { bgcolor: 'rgba(244, 67, 54, 0.14)' },
} as const
