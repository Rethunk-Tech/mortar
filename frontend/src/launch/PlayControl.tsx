import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Square } from 'lucide-react'
import { useState } from 'react'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { SetDefaultLaunchPreset } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileDialog } from '../game/profileDialog.ts'
import { useGameLoader } from '../games/info.ts'
import { formatElapsed } from '../i18n/formatElapsed.ts'
import { i18n } from '../i18n/index.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { useSecondValue } from '../i18n/useNow.ts'
import { useLoader } from '../loader/store.ts'
import { BASE_PRESET, playPresets } from '../profiles/profilePresets.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { usePlayDirect } from './directPref.ts'
import { PLAY_HEIGHT_PX } from './playHeight.ts'
import { StopDialog } from './StopDialog.tsx'
import { useLaunch } from './store.ts'
import { VanillaPlay } from './VanillaPlay.tsx'

function stateOf(status: Status | null, game: string): State {
  if (status?.game === game) {
    return status.state
  }
  return State.Idle
}

// The open profile's play presets from its two preset fields, so a save that changes neither leaves Play alone.
function useOpenPresets() {
  const launchPresets = useProfiles((s) => openProfileOf(s)?.launchPresets ?? null)
  const defaultLaunchPreset = useProfiles((s) => openProfileOf(s)?.defaultLaunchPreset ?? '')
  return playPresets({ launchPresets, defaultLaunchPreset })
}

// Play's label says what holds it back: mods updating first, else the loader installing.
function playLabel(installing: string, updating: number): string {
  if (updating > 0) {
    return plural({ count: updating }, { one: 'Updating # mod…', other: 'Updating # mods…' })
  }
  return installing ? i18n._(msg`Installing ${installing}…`) : i18n._(msg`Play`)
}

// RailStop is Stop as the narrow sidebar's full-width icon; its tooltip also says how long the game has run.
function RailStop({
  time,
  stopping,
  onStop,
}: {
  time: string
  stopping: boolean
  onStop: () => void
}) {
  const { t } = useLingui()
  return (
    <TipIconButton
      label={t`Stop game`}
      tip={
        <>
          {t`Stop game`}
          <br />
          {t`Running for ${time}`}
        </>
      }
      size="medium"
      fill={true}
      color="error"
      disabled={stopping}
      onClick={onStop}
      sx={{ width: '100%', height: PLAY_HEIGHT_PX, borderRadius: 0 }}
    >
      <Square size={24} />
    </TipIconButton>
  )
}

// `rail` is the narrow sidebar: Play and Stop shrink to icons.
export function PlayControl({ game, rail }: { game: string; rail: boolean }) {
  const { t } = useLingui()
  const loaderName = useGameLoader(game) || lowerFirst(t`Mod loader`)
  const status = useLaunch((s) => s.status)
  const stopping = useLaunch((s) => s.stopping)
  const updating = useLaunch((s) => s.updating)
  const starting = useLaunch((s) => s.starting)
  const installingLoader = useLoader((s) => s.installing)
  const start = useLaunch((s) => s.start)
  const openId = useProfiles((s) => s.openId)
  const [confirming, setConfirming] = useState(false)
  const state = stateOf(status, game)
  const running = state === State.Running
  const since = status?.since ?? 0
  const time = useSecondValue((now) => (running ? formatElapsed(now - since) : ''), running)
  // Fields, not the profiles: a save of the open profile re-renders this only when what it shows changed.
  const runningName = useProfiles((s) => s.profiles.find((p) => p.id === status?.profile)?.name)
  const presets = useOpenPresets()
  const direct = usePlayDirect()

  if (running) {
    const who = runningName !== undefined && status?.profile !== openId ? runningName : ''
    return (
      <>
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
          {rail ? null : (
            <Box
              title={runningName}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: space.gap,
                px: space.pad,
                py: space.gap,
              }}
            >
              <Box
                aria-hidden={true}
                sx={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  bgcolor: 'success.main',
                  flexShrink: 0,
                }}
              />
              <Typography
                role="status"
                noWrap={true}
                sx={{ fontSize: 'medium.fontSize', fontWeight: 600, minWidth: 0, flex: 1 }}
              >
                {who === '' ? t`Running` : t`Running ${who}`}
              </Typography>
              <Typography
                noWrap={true}
                sx={{
                  fontSize: 'lead.fontSize',
                  fontWeight: 600,
                  fontVariantNumeric: 'tabular-nums',
                  color: 'var(--mortar-ink-soft)',
                }}
              >
                {time}
              </Typography>
            </Box>
          )}
          {rail ? (
            <RailStop time={time} stopping={stopping} onStop={() => setConfirming(true)} />
          ) : (
            <Button
              variant="contained"
              color="error"
              fullWidth={true}
              disabled={stopping}
              startIcon={<Square size={18} fill="currentColor" />}
              onClick={() => setConfirming(true)}
              sx={{ height: PLAY_HEIGHT_PX, borderRadius: 0, fontSize: 20, fontWeight: 700 }}
            >
              {t`Stop game`}
            </Button>
          )}
        </Box>
        <StopDialog open={confirming} game={game} onClose={() => setConfirming(false)} />
      </>
    )
  }

  const busy = starting || installingLoader
  const launching = state === State.Launching
  const label = playLabel(installingLoader ? loaderName : '', updating)
  return (
    <VanillaPlay
      game={game}
      playDisabled={openId === '' || launching || busy}
      vanillaDisabled={launching || busy}
      label={label}
      rail={rail}
      play={() => start(game, openId, direct, '')}
      presets={presets}
      playWith={(key) => start(game, openId, direct, key)}
      launchOptions={openId === '' ? undefined : () => openProfileDialog('editLaunch', openId)}
      setDefault={(key) =>
        SetDefaultLaunchPreset(game, openId, key === BASE_PRESET ? '' : key)
          .then(useProfiles.getState().replace)
          .catch(reportError(t`Could not set the default launch preset`))
      }
    />
  )
}
