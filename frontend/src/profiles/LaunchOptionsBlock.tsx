import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { System } from '@wailsio/runtime'
import { useState } from 'react'
import { TestLaunch } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { STEAM } from '../brand/productNames.ts'
import { useGameBusy } from '../launch/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { TextInput } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
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
    <Box
      sx={{
        mt: space.gap,
        mb: space.gap,
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        flexWrap: 'wrap',
      }}
    >
      <DisabledReason title={t`Stop the game to test a launch.`} disabled={playing}>
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
                const outcome = await TestLaunch(gameId, profileId, '')
                if (outcome.reachedTitle) {
                  setResult(t`Reached the title screen`)
                  return
                }
                setResult(t`Crashed: ${{ cause: outcome.cause }}`)
              } catch (error) {
                setResult(errorMessage(error))
              }
            }
            run()
              .catch(reportUnexpected)
              .finally(() => setBusy(false))
          }}
        >{t`Test launch`}</Button>
      </DisabledReason>
      <Typography
        sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}
      >{t`Uses the saved options`}</Typography>
      {result ? (
        <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary', mt: space.gap }}>
          {result}
        </Typography>
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
      <TextInput
        label={t`Launch options`}
        value={launchOptions}
        onChange={(v) => {
          onLaunchOptions(v)
          onLaunchError(null)
        }}
        {...(launchError?.field === 'options' ? { error: launchError.message } : {})}
        hint={t`Extra launch arguments for this profile. Mortar already adds the ones the mod loader needs.`}
      />
      {/* Windows launches ignore a prefix, so there is nothing to set there. */}
      {System.IsWindows() ? null : (
        <TextInput
          label={t`Launch prefix`}
          value={launchPrefix}
          onChange={(v) => {
            onLaunchPrefix(v)
            onLaunchError(null)
          }}
          hint={t`Prefix for direct launches only (for example, gamemoderun mangohud).`}
        />
      )}
      <TextInput
        label={t`Launch environment`}
        value={launchEnv}
        multiline={true}
        onChange={(v) => {
          onLaunchEnv(v)
          onLaunchError(null)
        }}
        {...(launchError?.field === 'settings' ? { error: launchError.message } : {})}
        hint={t`One VAR=value per line, for direct launches only; ${{ launcher: STEAM }} launches ignore them.`}
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
