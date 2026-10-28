import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { useProfileScope } from './useProfileScope.ts'

/** Settings every profile of the game shares, held on the open profile's scope with a line saying so, so a change
 * there is never mistaken for this profile's own. */
export function SharedOnProfile({ children }: { children: ReactNode }) {
  const { t } = useLingui()
  const profile = useProfileScope()
  if (!profile) {
    return children
  }
  // Contents, so each wrapped setting stays its own tile inside a section rather than one tile holding them all.
  return (
    <Box className="settings-contents" sx={{ display: 'contents' }}>
      <Typography
        className="settings-caption"
        sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}
      >
        {t`Every profile of this game shares these. Switch to all profiles to change them.`}
      </Typography>
      {/* The tile keeps its surface; only what is on it fades. */}
      <Box
        className="settings-contents"
        inert={true}
        sx={{ display: 'contents', '& > * > *': { opacity: 0.6 } }}
      >
        {children}
      </Box>
    </Box>
  )
}
