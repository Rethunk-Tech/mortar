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
import { useEffect } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { useLaunch } from './store.ts'

const VISIBLE_LINES = 8
const LINE_HEIGHT = 20

function Overlay({ game }: { game: string }) {
  const { t } = useLingui()
  const name = useProfiles((s) => s.game?.name ?? '')
  const lines = useLaunch((s) => s.lines)
  const hide = useLaunch((s) => s.hide)
  const status = useLaunch((s) => s.status)
  const hidden = useLaunch((s) => s.hidden)
  if (status?.game !== game || status.state !== State.Launching || hidden) {
    return null
  }
  return (
    <Box
      role="dialog"
      aria-modal={true}
      aria-label={t`Launching ${name}`}
      sx={{
        position: 'absolute',
        inset: 0,
        zIndex: 5,
        bgcolor: 'rgba(0,0,0,0.80)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 2,
        p: 3,
      }}
    >
      <CircularProgress />
      <Typography sx={{ fontSize: 20, fontWeight: 700 }}>{t`Launching ${name}`}</Typography>
      <Box
        sx={{
          width: '100%',
          maxWidth: 640,
          minHeight: VISIBLE_LINES * LINE_HEIGHT,
          fontFamily: 'monospace',
          fontSize: 12,
          lineHeight: `${LINE_HEIGHT}px`,
          color: 'text.secondary',
          userSelect: 'text',
        }}
      >
        {lines.slice(-VISIBLE_LINES).map((line) => (
          <Typography key={line.id} noWrap={true} sx={{ font: 'inherit' }}>
            {line.text}
          </Typography>
        ))}
      </Box>
      <Button variant="outlined" onClick={hide} sx={{ whiteSpace: 'nowrap' }}>
        {t`Hide`}
      </Button>
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
      <DirectDialog />
    </>
  )
}
