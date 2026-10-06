import { Box } from '@mui/material'

// StableLabel shows one of several labels in the width of the widest, so a button does not resize as its state
// changes, in any language.
export function StableLabel({ labels, shown }: { labels: string[]; shown: string }) {
  return (
    <Box component="span" sx={{ display: 'inline-grid' }}>
      {labels.map((label) => (
        <Box
          key={label}
          component="span"
          aria-hidden={label !== shown}
          sx={{ gridArea: '1 / 1', visibility: label === shown ? 'visible' : 'hidden' }}
        >
          {label}
        </Box>
      ))}
    </Box>
  )
}
