import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Typography,
} from '@mui/material'
import { CircleAlert } from 'lucide-react'
import { useEffect } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { useLaunch } from './store.ts'

const VISIBLE_LINES = 8
const LINE_HEIGHT = 21
const LOG_PAD = 12
const SPINNER = 56
const SPINNER_THICKNESS = 3.9
const FULL = 100

const scrim = {
  position: 'absolute',
  inset: 0,
  zIndex: 5,
  bgcolor: 'rgba(0,0,0,0.80)',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  p: 3,
} as const

function Spinner() {
  return (
    <Box sx={{ position: 'relative', width: SPINNER, height: SPINNER }}>
      <CircularProgress
        variant="determinate"
        value={FULL}
        size={SPINNER}
        thickness={SPINNER_THICKNESS}
        sx={{ position: 'absolute', color: 'rgba(255,255,255,0.15)' }}
      />
      <CircularProgress size={SPINNER} thickness={SPINNER_THICKNESS} />
    </Box>
  )
}

function Overlay({ game }: { game: string }) {
  const { t } = useLingui()
  const name = useProfiles((s) => s.game?.name ?? '')
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const lines = useLaunch((s) => s.lines)
  const hide = useLaunch((s) => s.hide)
  const status = useLaunch((s) => s.status)
  const hidden = useLaunch((s) => s.hidden)
  if (status?.game !== game || status.state !== State.Launching || hidden) {
    return null
  }
  const profileName = profile?.name ?? ''
  const mods = (profile?.entries ?? []).reduce((n, e) => n + (e.mods ?? []).length, 0)
  const shown = lines.slice(-VISIBLE_LINES)
  return (
    <Box
      role="dialog"
      aria-modal={true}
      aria-label={t`Launching ${name}`}
      sx={{ ...scrim, gap: 2 }}
    >
      <Spinner />
      <Typography sx={{ fontSize: 30, fontWeight: 700 }}>{t`Launching ${name}`}</Typography>
      <Typography sx={{ fontSize: 16 }}>
        {t`${profileName} · ${plural(mods, { one: '# mod', other: '# mods' })}`}
      </Typography>
      <Box
        role="log"
        sx={{
          width: '100%',
          maxWidth: 720,
          boxSizing: 'border-box',
          minHeight: VISIBLE_LINES * LINE_HEIGHT + 2 * LOG_PAD,
          px: 1.75,
          py: 1.5,
          bgcolor: 'rgba(0,0,0,0.55)',
          fontFamily: 'monospace',
          fontSize: 13,
          lineHeight: `${LINE_HEIGHT}px`,
          color: 'text.secondary',
          userSelect: 'text',
        }}
      >
        {shown.map((line, i) => (
          <Typography
            key={line.id}
            noWrap={true}
            sx={{ font: 'inherit', color: i === shown.length - 1 ? '#ffffff' : 'inherit' }}
          >
            {line.text}
          </Typography>
        ))}
      </Box>
      <Button variant="outlined" onClick={hide}>
        {t`Hide`}
      </Button>
    </Box>
  )
}

function Failure({ game }: { game: string }) {
  const { t } = useLingui()
  const name = useProfiles((s) => s.game?.name ?? '')
  const failure = useLaunch((s) => s.failure)
  const dismiss = useLaunch((s) => s.dismissFailure)
  const start = useLaunch((s) => s.start)
  if (!failure) {
    return null
  }
  return (
    <Box sx={scrim}>
      <Box
        role="alert"
        sx={{
          width: '100%',
          maxWidth: 620,
          boxSizing: 'border-box',
          display: 'flex',
          flexDirection: 'column',
          gap: 1.75,
          p: 3,
          bgcolor: 'rgba(38,38,46,0.98)',
          border: '1px solid rgba(255,138,128,0.5)',
          borderRadius: '8px',
        }}
      >
        <Box
          sx={{ display: 'flex', alignItems: 'center', gap: 1.25, fontSize: 22, fontWeight: 700 }}
        >
          <CircleAlert size={22} color="#ff9a90" aria-hidden={true} />
          {t`${name} did not start`}
        </Box>
        <Typography sx={{ fontSize: 15, lineHeight: 1.5 }}>{failure.body}</Typography>
        <Box sx={{ display: 'flex', gap: 1.25, justifyContent: 'flex-end' }}>
          <Button variant="outlined" onClick={dismiss}>
            {t`Close`}
          </Button>
          <Button
            variant="contained"
            onClick={() => {
              dismiss()
              start(game, failure.profile, false)
            }}
          >
            {t`Try again`}
          </Button>
        </Box>
      </Box>
    </Box>
  )
}

function DirectDialog() {
  const { t } = useLingui()
  const ask = useLaunch((s) => s.askDirect)
  const answer = useLaunch((s) => s.answerDirect)
  return (
    <Dialog
      open={ask !== null}
      onClose={() => answer(false)}
      slotProps={{ paper: { sx: { bgcolor: 'rgba(40,40,48,0.92)', maxWidth: 440 } } }}
    >
      <DialogTitle>{t`Steam was not found`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`Mortar can start SMAPI directly instead. The Steam overlay and Steam's playtime tracking will not work while you play this way.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => answer(false)} sx={{ whiteSpace: 'nowrap' }}>
          {t`Cancel`}
        </Button>
        <Button variant="contained" onClick={() => answer(true)} sx={{ whiteSpace: 'nowrap' }}>
          {t`Launch without Steam`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export function LaunchLayer({ game }: { game: string }) {
  const refresh = useLaunch((s) => s.refresh)
  useEffect(() => {
    refresh(game)
  }, [game, refresh])
  return (
    <>
      <Overlay game={game} />
      <Failure game={game} />
      <DirectDialog />
    </>
  )
}
