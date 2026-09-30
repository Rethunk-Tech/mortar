import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Typography } from '@mui/material'
import { Square } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { compact } from '../game/compact.ts'
import { useLoader } from '../loader/store.ts'
import { useProfiles } from '../profiles/store.ts'
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

export function PlayControl({ game }: { game: string }) {
  const { t } = useLingui()
  const status = useLaunch((s) => s.status)
  const stopping = useLaunch((s) => s.stopping)
  const starting = useLaunch((s) => s.starting)
  const installingLoader = useLoader((s) => s.installing)
  const start = useLaunch((s) => s.start)
  const openId = useProfiles((s) => s.openId)
  const profiles = useProfiles((s) => s.profiles)
  const [confirming, setConfirming] = useState(false)
  const state = stateOf(status, game)
  const running = state === State.Running
  const time = useElapsed(status?.since ?? 0, running)
  const runningProfile = profiles.find((p) => p.id === status?.profile)

  if (running) {
    return (
      <>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: 0.75,
            p: 1,
            [compact]: { alignItems: 'center', p: 0 },
          }}
        >
          <Box
            role="status"
            title={runningProfile?.name}
            sx={{
              display: 'flex',
              alignItems: 'baseline',
              justifyContent: 'space-between',
              gap: 1,
              px: 0.5,
              [compact]: { display: 'none' },
            }}
          >
            <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 700, color: 'success.main' }}>
              {t`Running`}
            </Typography>
            <Typography noWrap={true} sx={{ fontSize: 14, fontVariantNumeric: 'tabular-nums' }}>
              {time}
            </Typography>
          </Box>
          {runningProfile && runningProfile.id !== openId ? (
            <Typography
              noWrap={true}
              sx={{
                px: 0.5,
                fontSize: 12,
                color: 'text.secondary',
                [compact]: { display: 'none' },
              }}
            >
              {t`Playing ${runningProfile.name}`}
            </Typography>
          ) : null}
          <Button
            variant="outlined"
            color="error"
            fullWidth={true}
            disabled={stopping}
            startIcon={<Square size={16} />}
            onClick={() => setConfirming(true)}
            sx={{ whiteSpace: 'nowrap', [compact]: { display: 'none' } }}
          >
            {t`Stop game`}
          </Button>
          <IconButton
            aria-label={t`Stop game`}
            title={t`Running ${time}`}
            color="error"
            disabled={stopping}
            onClick={() => setConfirming(true)}
            sx={{ display: 'none', [compact]: { display: 'inline-flex' } }}
          >
            <Square size={20} />
          </IconButton>
        </Box>
        <StopDialog open={confirming} game={game} onClose={() => setConfirming(false)} />
      </>
    )
  }

  const busy = starting || installingLoader
  const launching = state === State.Launching
  return (
    <VanillaPlay
      game={game}
      playDisabled={openId === '' || launching || busy}
      vanillaDisabled={launching || busy}
      label={installingLoader ? t`Installing SMAPI…` : t`Play`}
      play={() => start(game, openId, false)}
    />
  )
}
