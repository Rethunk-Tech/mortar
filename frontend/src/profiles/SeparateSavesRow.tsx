import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel, Switch, Typography } from '@mui/material'
import { useState } from 'react'
import { SetSeparateSaves } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useGameInfo } from '../games/info.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useProfiles } from './store.ts'

// One switch in profile settings. Turning it on asks first, because Mortar then sets the game's shared saves aside
// while this profile runs and puts them back afterwards; the shared saves are never moved for good or deleted.
export function SeparateSavesRow({
  gameId,
  profileId,
  on,
}: {
  gameId: string
  profileId: string
  on: boolean
}) {
  const { t } = useLingui()
  const replace = useProfiles((s) => s.replace)
  const hasSaves = useGameInfo(gameId)?.hasSaves ?? false
  const [asking, setAsking] = useState(false)
  const [copyIn, setCopyIn] = useState(true)
  if (!hasSaves) {
    return null
  }
  const apply = (next: boolean, copy: boolean) => {
    SetSeparateSaves(gameId, profileId, next, copy)
      .then((p) => p && replace(p))
      .catch(reportUnexpected)
  }
  return (
    <Box sx={{ mt: 1 }}>
      <FormControlLabel
        control={
          <Switch
            checked={on}
            onChange={(event) => (event.target.checked ? setAsking(true) : apply(false, false))}
          />
        }
        label={t`Keep this profile's saves separate`}
      />
      {on ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`This profile plays on its own saves. Your shared saves are untouched.`}
        </Typography>
      ) : null}
      <ConfirmDialog
        open={asking}
        title={t`Keep this profile's saves separate?`}
        body={t`While this profile runs, Mortar sets your shared saves aside and shows the game the profile's own folder. When the game closes, your shared saves come back as they were. Nothing is deleted.`}
        confirmLabel={t`Enable`}
        onCancel={() => setAsking(false)}
        onConfirm={() => {
          setAsking(false)
          apply(true, copyIn)
        }}
      >
        <FormControlLabel
          control={<Checkbox checked={copyIn} onChange={(e) => setCopyIn(e.target.checked)} />}
          label={t`Start with a copy of my current saves`}
        />
      </ConfirmDialog>
    </Box>
  )
}
