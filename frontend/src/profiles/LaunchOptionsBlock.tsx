import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { useState } from 'react'
import { TestLaunch } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { useGameBusy } from '../launch/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { LaunchPreview } from './LaunchPreview.tsx'
import { useProfiles } from './store.ts'

type LaunchError = { field: 'options' | 'settings'; message: string } | null

function TestLaunchRow({
  gameId,
  profileId,
  launchOptions,
  launchPrefix,
  launchEnv,
  onLaunchError,
}: {
  gameId: string
  profileId: string
  launchOptions: string
  launchPrefix: string
  launchEnv: string
  onLaunchError: (value: LaunchError) => void
}) {
  const { t } = useLingui()
  const setLaunchOptions = useProfiles((s) => s.setLaunchOptions)
  const setLaunchSettings = useProfiles((s) => s.setLaunchSettings)
  const playing = useGameBusy()
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState('')
  return (
    <Box sx={{ mt: 1, mb: 1, display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
      <Button
        disabled={!(gameId && !playing && !busy)}
        onClick={() => {
          setBusy(true)
          setResult('')
          onLaunchError(null)
          const run = async () => {
            try {
              await setLaunchOptions(profileId, launchOptions)
            } catch (error) {
              onLaunchError({ field: 'options', message: errorMessage(error) })
              return
            }
            try {
              await setLaunchSettings(profileId, launchPrefix, launchEnv)
            } catch (error) {
              onLaunchError({ field: 'settings', message: errorMessage(error) })
              return
            }
            try {
              const outcome = await TestLaunch(gameId, profileId)
              if (outcome.reachedTitle) {
                setResult(t`Reached the title screen`)
                return
              }
              setResult(t`Crashed: ${outcome.cause}`)
            } catch (error) {
              setResult(errorMessage(error))
            }
          }
          run()
            .catch(reportUnexpected)
            .finally(() => setBusy(false))
        }}
      >{t`Test launch`}</Button>
      <Typography
        sx={{ fontSize: 13, color: 'text.secondary' }}
      >{t`Uses the saved options`}</Typography>
      {result ? (
        <Typography sx={{ fontSize: 13, color: 'text.secondary', mt: 1 }}>{result}</Typography>
      ) : null}
    </Box>
  )
}

function LaunchOptionsBlock({
  gameId,
  profileId,
  launchOptions,
  onLaunchOptions,
  launchPrefix,
  onLaunchPrefix,
  launchEnv,
  onLaunchEnv,
  launchError,
  onLaunchError,
}: {
  gameId: string
  profileId: string
  launchOptions: string
  onLaunchOptions: (value: string) => void
  launchPrefix: string
  onLaunchPrefix: (value: string) => void
  launchEnv: string
  onLaunchEnv: (value: string) => void
  launchError: LaunchError
  onLaunchError: (value: LaunchError) => void
}) {
  const { t } = useLingui()
  return (
    <>
      <TextField
        fullWidth={true}
        margin="dense"
        label={t`Launch options`}
        value={launchOptions}
        onChange={(event) => {
          onLaunchOptions(event.target.value)
          onLaunchError(null)
        }}
        error={launchError?.field === 'options'}
        helperText={
          (launchError?.field === 'options' && launchError.message) ||
          t`Extra SMAPI arguments for this profile. Mortar already sets the mods folder.`
        }
        slotProps={{ root: { sx: { userSelect: 'text' } } }}
      />
      <TextField
        fullWidth={true}
        margin="dense"
        label={t`Launch prefix`}
        value={launchPrefix}
        onChange={(event) => {
          onLaunchPrefix(event.target.value)
          onLaunchError(null)
        }}
        error={launchError?.field === 'settings'}
        helperText={t`Prefix for direct launches only (for example, gamemoderun mangohud). On Windows, prefixes are unavailable.`}
        slotProps={{ root: { sx: { userSelect: 'text' } } }}
      />
      <TextField
        fullWidth={true}
        margin="dense"
        multiline={true}
        minRows={2}
        label={t`Launch environment`}
        value={launchEnv}
        onChange={(event) => {
          onLaunchEnv(event.target.value)
          onLaunchError(null)
        }}
        error={launchError?.field === 'settings'}
        helperText={
          (launchError?.field === 'settings' && launchError.message) ||
          t`One VAR=value per line; applies to direct launches only. Steam launches do not receive these settings.`
        }
        slotProps={{ root: { sx: { userSelect: 'text' } } }}
      />
      <LaunchPreview
        gameId={gameId}
        profileId={profileId}
        options={launchOptions}
        prefix={launchPrefix}
        env={launchEnv}
      />
      <TestLaunchRow
        gameId={gameId}
        profileId={profileId}
        launchOptions={launchOptions}
        launchPrefix={launchPrefix}
        launchEnv={launchEnv}
        onLaunchError={onLaunchError}
      />
    </>
  )
}

export { LaunchOptionsBlock }
