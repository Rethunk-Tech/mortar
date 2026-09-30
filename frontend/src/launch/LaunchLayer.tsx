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
import { Clipboard } from '@wailsio/runtime'
import { CircleAlert, Copy } from 'lucide-react'
import { useEffect } from 'react'
import { Hint } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { format } from '../console/filter.ts'
import { useConsole } from '../console/store.ts'
import { launchLine } from '../firstrun/logic.ts'
import { useTab } from '../game/tab.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
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

// While the overlay is up, what it covers is inert so focus cannot reach it; a stable function so the ref runs once.
function holdFocus(el: HTMLElement) {
  const covered = [...(el.parentElement?.children ?? [])].filter(
    (c): c is HTMLElement => c !== el && c instanceof HTMLElement && !c.inert,
  )
  for (const c of covered) {
    c.inert = true
  }
  return () => {
    for (const c of covered) {
      c.inert = false
    }
  }
}

function Overlay({ game }: { game: string }) {
  const { t } = useLingui()
  const name = useProfiles((s) => s.game?.name ?? '')
  const status = useLaunch((s) => s.status)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === status?.profile))
  const entries = useConsole((s) => s.entries)
  const setTab = useTab((s) => s.setTab)
  const hide = useLaunch((s) => s.hide)
  const hidden = useLaunch((s) => s.hidden)
  if (status?.game !== game || status.state !== State.Launching || hidden) {
    return null
  }
  const profileName = profile?.name ?? ''
  const mods = profile ? userModCount(profile) : 0
  const shown = entries.slice(-VISIBLE_LINES)
  return (
    <Box
      ref={holdFocus}
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
            key={line.seq}
            noWrap={true}
            sx={{ font: 'inherit', color: i === shown.length - 1 ? '#ffffff' : 'inherit' }}
          >
            {format(line)}
          </Typography>
        ))}
      </Box>
      <Box sx={{ display: 'flex', gap: 1.25 }}>
        <Button
          variant="outlined"
          autoFocus={true}
          onClick={() => {
            setTab('console')
            hide()
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Open console`}
        </Button>
        <Button variant="outlined" onClick={hide}>
          {t`Hide`}
        </Button>
      </Box>
    </Box>
  )
}

function LaunchLine({ line }: { line: string }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', gap: 1 }}>
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          alignItems: 'center',
          px: 1.5,
          py: 0.75,
          bgcolor: 'rgba(0,0,0,0.45)',
          border: '1px solid rgba(255,255,255,0.15)',
          borderRadius: '6px',
          fontFamily: 'monospace',
          fontSize: 13,
          wordBreak: 'break-all',
          userSelect: 'text',
        }}
      >
        {line}
      </Box>
      <Button
        variant="outlined"
        startIcon={<Copy size={16} />}
        onClick={() => {
          Clipboard.SetText(line).then(
            () => useToasts.getState().push({ kind: 'success', title: t`Launch options copied` }),
            reportUnexpected,
          )
        }}
        sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Copy`}
      </Button>
    </Box>
  )
}

function Failure({ game }: { game: string }) {
  const { t } = useLingui()
  const info = useProfiles((s) => s.game)
  const failure = useLaunch((s) => s.failure)
  const dismiss = useLaunch((s) => s.dismissFailure)
  const start = useLaunch((s) => s.start)
  // The dialog unmounts with `failure`, so nothing fades out with stale text.
  if (!failure) {
    return null
  }
  const name = info?.name ?? ''
  const showLine = failure.hint === Hint.HintLaunchOptions && info?.installDir
  return (
    <Dialog
      open={true}
      onClose={dismiss}
      fullWidth={true}
      maxWidth="sm"
      slotProps={{ paper: { sx: { bgcolor: 'rgb(38,38,46)' } } }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
        <CircleAlert size={22} color="#ff9a90" aria-hidden={true} />
        {t`${name} did not start`}
      </DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1.75 }}>
        <DialogContentText>{failure.body}</DialogContentText>
        {showLine ? <LaunchLine line={launchLine(info.installDir)} /> : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={dismiss}>{t`Close`}</Button>
        <Button
          variant="contained"
          onClick={() => {
            dismiss()
            start(game, failure.profile, false)
          }}
        >
          {t`Try again`}
        </Button>
      </DialogActions>
    </Dialog>
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
      slotProps={{ paper: { sx: { bgcolor: 'rgb(40,40,48)', maxWidth: 440 } } }}
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
