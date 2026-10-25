import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { useSettingsScope } from './settingsScope.ts'

/** Settings every profile of the game shares, held on the open profile's scope with a line saying so, so a change
 * there is never mistaken for this profile's own. */
export function SharedOnProfile({ children }: { children: ReactNode }) {
  const { t } = useLingui()
  const profile = useSettingsScope((s) => s.profile)
  if (!profile) {
    return children
  }
  return (
    <Box>
      <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary', mb: 1 }}>
        {t`Every profile of this game shares these. Switch to all profiles to change them.`}
      </Typography>
      <Box inert={true} sx={{ opacity: 0.6 }}>
        {children}
      </Box>
    </Box>
  )
}
