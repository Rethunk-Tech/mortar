import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { type ReactNode, useState } from 'react'
import { useGameInfo } from '../games/info.ts'
import { EditProfileDialog } from '../profiles/EditProfileDialog.tsx'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { matchPerProfile } from './perProfileMatch.ts'
import type { SettingsShell } from './SettingsShell.tsx'

type Related = NonNullable<Parameters<typeof SettingsShell>[0]['related']>

// Zoom, UI scale and the other startup settings are kept per profile, in Edit profile's Game settings tab. A game
// settings search for one shows a row that opens it there for the open profile; `dialog` is that Edit profile.
export function usePerProfileRelated(game: string): { related?: Related; dialog: ReactNode } {
  const { t } = useLingui()
  const startup = useGameInfo(game)?.startupSettings === true
  const profile = useProfiles(openProfileOf)
  const [editing, setEditing] = useState(false)
  if (!(startup && profile)) {
    return { dialog: null }
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
    related: {
      match: (query) => matchPerProfile(query, labels),
      row: (label) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
          <Typography sx={{ flex: 1, fontSize: 14 }}>{t`${label} is set per profile`}</Typography>
          <Button onClick={() => setEditing(true)}>{t`Edit ${profile.name}`}</Button>
        </Box>
      ),
    },
    dialog: (
      <EditProfileDialog
        profile={profile}
        open={editing}
        onClose={() => setEditing(false)}
        initialTab="game"
      />
    ),
  }
}
