import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useGameInfo } from '../games/info.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { matchPerProfile } from './perProfileMatch.ts'
import type { SettingsShell } from './SettingsShell.tsx'

type Related = NonNullable<Parameters<typeof SettingsShell>[0]['related']>

// Zoom, UI scale and the other startup settings are kept per profile, on this page's profile scope. A game settings
// search for one shows a row that switches to it for the open profile; show does the switch.
export function usePerProfileRelated(game: string, show: () => void): Related | undefined {
  const { t } = useLingui()
  const startup = useGameInfo(game)?.startupSettings === true
  const profile = useProfiles(openProfileOf)
  if (!(startup && profile)) {
    return undefined
  }
  const labels = [
    t`Window mode`,
    t`Resolution`,
    t`Zoom`,
    t`UI scale`,
    t`Music volume`,
    t`Sound volume`,
    t`Start muted`,
  ]
  return {
    match: (query) => matchPerProfile(query, labels),
    row: (label) => (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
        <Typography
          sx={{ flex: 1, fontSize: 'medium.fontSize' }}
        >{t`${{ name: label }} is set per profile`}</Typography>
        <Button onClick={show}>{t`Show for ${{ name: profile.name }}`}</Button>
      </Box>
    ),
  }
}
