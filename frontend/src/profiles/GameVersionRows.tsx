import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { useState } from 'react'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import {
  CanGetOlderVersion,
  SetGameFolder,
  SetModsUpTo,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { Field, TextInput } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { OlderVersionDialog } from './OlderVersionDialog.tsx'
import { useProfiles } from './store.ts'

const FOLDER_PIN = 'dir:'
const hint = { fontSize: 'small.fontSize', color: 'text.secondary' } as const

// The profile's game folder (an old copy of the game) and its mod date limit, each saved as it is changed.
export function GameVersionRows({
  gameId,
  profileId,
  install,
  modsUpTo,
}: {
  gameId: string
  profileId: string
  install: string
  modsUpTo: string
}) {
  const { t } = useLingui()
  const replace = useProfiles((s) => s.replace)
  const [asking, setAsking] = useState(false)
  const { data: steam } = useLoaded(() => CanGetOlderVersion(gameId), [gameId], false)
  const folder = install.startsWith(FOLDER_PIN) ? install.slice(FOLDER_PIN.length) : ''
  const setFolder = (dir: string) =>
    SetGameFolder(gameId, profileId, dir)
      .then((p) => p && replace(p))
      .catch(reportError(t`Could not use that folder`))
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, mt: space.gap }}>
      <Field label={t`Game folder`}>
        <Typography sx={{ overflowWrap: 'anywhere' }}>
          {folder || t`The selected install`}
        </Typography>
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap, ml: -1 }}>
          <Button
            onClick={() => {
              PickFolder(t`Choose the game folder`)
                .then((dir) => {
                  if (dir) {
                    setFolder(dir)
                  }
                })
                .catch(reportUnexpected)
            }}
          >
            {t`Choose folder…`}
          </Button>
          {folder ? (
            <Button onClick={() => setFolder('')}>{t`Use the selected install`}</Button>
          ) : null}
          {steam ? (
            <Button onClick={() => setAsking(true)}>{t`Get an older version…`}</Button>
          ) : null}
        </Box>
        <Typography sx={hint}>
          {t`This profile runs the game from this folder, an older copy, instead of the selected install.`}
        </Typography>
      </Field>
      <TextInput
        label={t`Mod versions up to`}
        type="date"
        value={modsUpTo}
        onChange={(date) => {
          SetModsUpTo(gameId, profileId, date)
            .then((p) => p && replace(p))
            .catch(reportError(t`Could not save the date`))
        }}
        hint={t`Installs from Browse and updates pick the newest version released on or before this date. Mods from sites that list no dates are not filtered.`}
      />
      <OlderVersionDialog
        gameId={gameId}
        profileId={profileId}
        open={asking}
        onClose={() => setAsking(false)}
      />
    </Box>
  )
}
