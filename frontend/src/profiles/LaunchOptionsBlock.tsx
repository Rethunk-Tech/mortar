import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select, TextField, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { TestLaunch } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { LaunchPreset } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  AddLaunchPreset,
  ListLaunchPresets,
  RemoveLaunchPreset,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useLaunch } from '../launch/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { applyLaunchPreset } from './applyLaunchPreset.ts'
import { LaunchPreview } from './LaunchPreview.tsx'
import { useProfiles } from './store.ts'

type LaunchError = { field: 'options' | 'settings'; message: string } | null

function LaunchPresetBar({
  gameId,
  launchOptions,
  launchPrefix,
  launchEnv,
  onFill,
}: {
  gameId: string
  launchOptions: string
  launchPrefix: string
  launchEnv: string
  onFill: (options: string, prefix: string, env: string) => void
}) {
  const { t } = useLingui()
  const [presets, setPresets] = useState<LaunchPreset[]>([])
  const [selected, setSelected] = useState('')
  const [saveName, setSaveName] = useState('')
  const reload = useCallback(() => {
    if (!gameId) {
      return
    }
    ListLaunchPresets(gameId)
      .then((next) => setPresets(next ?? []))
      .catch(reportUnexpected)
  }, [gameId])
  useEffect(() => {
    reload()
  }, [reload])
  const filled = applyLaunchPreset({
    options: launchOptions,
    prefix: launchPrefix,
    env: launchEnv,
  })
  return (
    <Box sx={{ mt: 1, mb: 1 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>{t`Preset`}</Typography>
      <Select
        displayEmpty={true}
        fullWidth={true}
        size="small"
        value={selected}
        onChange={(event) => {
          const name = event.target.value
          setSelected(name)
          const preset = presets.find((item) => item.name === name)
          if (preset) {
            const next = applyLaunchPreset(preset)
            onFill(next.options, next.prefix, next.env)
          }
        }}
        sx={{ mb: 1, userSelect: 'text' }}
      >
        <MenuItem value="">{t`None`}</MenuItem>
        {presets.map((preset) => (
          <MenuItem key={preset.name} value={preset.name ?? ''}>
            {preset.name}
          </MenuItem>
        ))}
      </Select>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
        <TextField
          size="small"
          label={t`Preset name`}
          value={saveName}
          onChange={(event) => setSaveName(event.target.value)}
          slotProps={{ root: { sx: { userSelect: 'text', flex: 1, minWidth: 120 } } }}
        />
        <Button
          disabled={!(gameId && saveName.trim())}
          onClick={() => {
            AddLaunchPreset(gameId, saveName.trim(), filled.options, filled.prefix, filled.env)
              .then(() => {
                setSelected(saveName.trim())
                reload()
              })
              .catch(reportUnexpected)
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >{t`Save as preset`}</Button>
        <Button
          disabled={!(gameId && selected)}
          onClick={() => {
            RemoveLaunchPreset(gameId, selected)
              .then(() => {
                setSelected('')
                reload()
              })
              .catch(reportUnexpected)
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >{t`Remove preset`}</Button>
      </Box>
    </Box>
  )
}

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
  const playing = useLaunch(
    (s) => s.starting || s.status?.state === State.Launching || s.status?.state === State.Running,
  )
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState('')
  return (
    <Box sx={{ mt: 1, mb: 1 }}>
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
        sx={{ whiteSpace: 'nowrap' }}
      >{t`Test launch`}</Button>
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
      <LaunchPresetBar
        gameId={gameId}
        launchOptions={launchOptions}
        launchPrefix={launchPrefix}
        launchEnv={launchEnv}
        onFill={(options, prefix, env) => {
          onLaunchOptions(options)
          onLaunchPrefix(prefix)
          onLaunchEnv(env)
          onLaunchError(null)
        }}
      />
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
