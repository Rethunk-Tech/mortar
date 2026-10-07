import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import type { VortexInventory } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/migrate/models.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { ExternalVortexContents } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useGameName } from '../games/info.ts'
import { reportUnexpected } from '../toasts/report.ts'

function useVortexNote(game: string, contents: VortexInventory | null): string {
  const { t } = useLingui()
  const gameName = useGameName(game)
  const folder = contents?.folder ?? ''
  const found = (contents?.games ?? []).map((g) => `${g.id} (${g.profiles})`).join(', ')
  if (!folder) {
    return t`Use Vortex from another folder?`
  }
  if (!found) {
    return t`The Vortex folder ${folder} has no profiles.`
  }
  return t`This Vortex folder has no profiles for ${gameName}: found ${found}.`
}

// Shown when no Vortex profile matches the game: says what the Vortex folder does hold, and lets the user
// point Mortar at another one.
export function VortexFolderPrompt({ game, onChosen }: { game: string; onChosen: () => void }) {
  const { t } = useLingui()
  const [contents, setContents] = useState<VortexInventory | null>(null)
  const [error, setError] = useState('')
  const load = useCallback(() => {
    ExternalVortexContents().then(setContents).catch(reportUnexpected)
  }, [])
  useEffect(load, [load])
  const note = useVortexNote(game, contents)
  const [typing, setTyping] = useState(false)
  const [typed, setTyped] = useState('')
  const apply = async (dir: string) => {
    try {
      await SetByKey('vortexFolder', dir, '')
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return
    }
    setError('')
    load()
    onChosen()
  }
  // Without a native folder dialog (server mode) the path is typed instead.
  const choose = () => {
    PickFolder(t`Vortex data folder`)
      .then((dir) => (dir ? apply(dir) : undefined))
      .catch(() => setTyping(true))
  }
  return (
    <Typography variant="body2" color="text.secondary" sx={{ pt: 1 }}>
      {note}{' '}
      <Button size="small" onClick={choose}>
        {t`Choose folder…`}
      </Button>
      {typing ? (
        <Box component="span" sx={{ display: 'flex', gap: 1, pt: 1 }}>
          <TextField
            size="small"
            fullWidth={true}
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            label={t`Vortex data folder`}
          />
          <Button disabled={!typed.trim()} onClick={() => apply(typed.trim())}>
            {t`Use folder`}
          </Button>
        </Box>
      ) : null}
      {error ? (
        <Typography component="span" variant="body2" color="error" role="alert" sx={{ ml: 1 }}>
          {error}
        </Typography>
      ) : null}
    </Typography>
  )
}
