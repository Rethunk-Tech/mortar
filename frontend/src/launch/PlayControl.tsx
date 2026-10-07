import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Tooltip, Typography } from '@mui/material'
import { Square } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { SetDefaultLaunchPreset } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useGameLoader } from '../games/info.ts'
import { useLoader } from '../loader/store.ts'
import { BASE_PRESET, playPresets } from '../profiles/profilePresets.ts'
import { useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { playDirect } from './directPref.ts'
import { useLive } from './live.ts'
import { PLAY_HEIGHT_PX } from './playHeight.ts'
import { StopDialog } from './StopDialog.tsx'
import { useLaunch } from './store.ts'
import { VanillaPlay } from './VanillaPlay.tsx'

const MS = 1000
const MINUTE = 60
const HOUR = 3600
const DIGITS = 2

function pad(n: number, width: number): string {
  return String(n).padStart(width, '0')
}

function elapsed(ms: number): string {
  const total = Math.max(0, Math.floor(ms / MS))
  const hours = Math.floor(total / HOUR)
  const minutes = Math.floor((total % HOUR) / MINUTE)
  const seconds = pad(total % MINUTE, DIGITS)
  if (hours > 0) {
    return `${hours}:${pad(minutes, DIGITS)}:${seconds}`
  }
  return `${minutes}:${seconds}`
}

function stateOf(status: Status | null, game: string): State {
  if (status?.game === game) {
    return status.state
  }
  return State.Idle
}

function useElapsed(since: number, active: boolean): string {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (!active) {
      return
    }
    const id = setInterval(() => setNow(Date.now()), MS)
    return () => clearInterval(id)
  }, [active])
  return elapsed(now - since)
}

// `rail` is the narrow sidebar: Play and Stop shrink to icons.
export function PlayControl({ game, rail }: { game: string; rail: boolean }) {
  const { t } = useLingui()
  const loaderName = useGameLoader(game) || t`the mod loader`
  const status = useLaunch((s) => s.status)
  const stopping = useLaunch((s) => s.stopping)
  const updating = useLaunch((s) => s.updating)
  const starting = useLaunch((s) => s.starting)
  const installingLoader = useLoader((s) => s.installing)
  const start = useLaunch((s) => s.start)
  const openId = useProfiles((s) => s.openId)
  const profiles = useProfiles((s) => s.profiles)
  const [confirming, setConfirming] = useState(false)
  const state = stateOf(status, game)
  const running = state === State.Running
  const time = useElapsed(status?.since ?? 0, running)
  const scene = useLive((s) => (s.game === game ? s.scene : ''))
  const runningProfile = profiles.find((p) => p.id === status?.profile)
  const replaceProfile = useProfiles((s) => s.replace)
  const presets = playPresets(profiles.find((p) => p.id === openId))

  if (running) {
    const who = runningProfile && runningProfile.id !== openId ? runningProfile.name : ''
    return (
      <>
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
          {rail ? null : (
            <Box
              title={runningProfile?.name}
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
                sx={{ fontSize: 14, fontWeight: 600, minWidth: 0, flex: 1 }}
              >
                {who === '' ? t`Running` : t`Running ${who}`}
              </Typography>
              {scene ? (
                <Typography
                  noWrap={true}
                  title={t`Current scene`}
                  sx={{ fontSize: 13, color: 'var(--mortar-ink-soft)', minWidth: 0 }}
                >
                  {scene}
                </Typography>
              ) : null}
              <Typography
                noWrap={true}
                sx={{
                  fontSize: 15,
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
            <Tooltip title={t`Stop game`} describeChild={true}>
              <span style={{ display: 'flex' }}>
                <IconButton
                  aria-label={t`Stop game`}
                  title={t`Running for ${time}`}
                  color="error"
                  disabled={stopping}
                  onClick={() => setConfirming(true)}
                  sx={{ width: '100%', height: PLAY_HEIGHT_PX, borderRadius: 0 }}
                >
                  <Square size={24} />
                </IconButton>
              </span>
            </Tooltip>
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
  let label = t`Play`
  if (installingLoader) {
    label = t`Installing ${loaderName}…`
  }
  if (updating > 0) {
    label = t`${plural(updating, { one: 'Updating # mod…', other: 'Updating # mods…' })}`
  }
  return (
    <VanillaPlay
      game={game}
      playDisabled={openId === '' || launching || busy}
      vanillaDisabled={launching || busy}
      label={label}
      rail={rail}
      play={() => start(game, openId, playDirect(), '')}
      presets={presets}
      playWith={(key) => start(game, openId, playDirect(), key)}
      setDefault={(key) =>
        SetDefaultLaunchPreset(game, openId, key === BASE_PRESET ? '' : key)
          .then(replaceProfile)
          .catch(reportError(t`Could not set the default launch preset`))
      }
    />
  )
}
