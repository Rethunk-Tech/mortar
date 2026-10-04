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
import { useTheme } from '@mui/material/styles'
import { Clipboard } from '@wailsio/runtime'
import { CircleAlert, Copy } from 'lucide-react'
import { useEffect } from 'react'
import { Hint } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { CrashDialog, SwitchOffButton } from '../console/CrashDialog.tsx'
import { format } from '../console/filter.ts'
import { useConsole } from '../console/store.ts'
import { launchLine } from '../firstrun/logic.ts'
import { useTab } from '../game/tab.ts'
import { nexusIdOf } from '../mods/lookup.ts'
import { openPage } from '../mods/menu.ts'
import { useMods } from '../mods/store.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { controlsCutout } from '../shell/controlsCutout.ts'
import { FlatpakGrant } from '../shell/FlatpakGrant.tsx'
import { MONO } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { holdFocus, launchEscHides } from './holdFocus.ts'
import { KnownGoodOffer } from './KnownGoodOffer.tsx'
import { PrePlayDialog } from './PrePlayDialog.tsx'
import { SaveWarnDialog } from './SaveWarnDialog.tsx'
import { useLaunch } from './store.ts'
import { UpdateWarnDialog } from './UpdateWarnDialog.tsx'
import { VanillaPlayDialogs } from './VanillaPlay.tsx'

const VISIBLE_LINES = 8
const LINE_HEIGHT = 21
const LOG_PAD = 12
const SPINNER = 56
const SPINNER_THICKNESS = 3.9
const FULL = 100

const scrim = {
  position: 'fixed',
  inset: 0,
  zIndex: 1200,
  bgcolor: 'var(--mortar-overlay-80)',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  p: 3,
  clipPath: controlsCutout,
} as const

function Spinner() {
  return (
    <Box sx={{ position: 'relative', width: SPINNER, height: SPINNER }}>
      <CircularProgress
        variant="determinate"
        value={FULL}
        size={SPINNER}
        thickness={SPINNER_THICKNESS}
        sx={{ position: 'absolute', color: 'var(--mortar-hairline-15)' }}
      />
      <CircularProgress size={SPINNER} thickness={SPINNER_THICKNESS} />
    </Box>
  )
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
  const overlayOpen = status?.game === game && status.state === State.Launching && !hidden
  useEffect(() => {
    if (!overlayOpen) {
      return
    }
    const onKey = (e: KeyboardEvent) => {
      if (!launchEscHides(e.key, document.querySelectorAll('[role="dialog"]').length)) {
        return
      }
      e.preventDefault()
      hide()
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [hide, overlayOpen])
  if (!overlayOpen) {
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
      <Box
        aria-hidden={true}
        sx={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 'var(--window-controls)',
          height: 'var(--title-bar)',
          '--wails-draggable': 'drag',
        }}
      />
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
          bgcolor: 'var(--mortar-overlay-55)',
          fontFamily: MONO,
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
            sx={{
              font: 'inherit',
              color: i === shown.length - 1 ? 'var(--mortar-ink)' : 'inherit',
            }}
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
          bgcolor: 'var(--mortar-overlay-45)',
          border: '1px solid var(--mortar-hairline-15)',
          borderRadius: '6px',
          fontFamily: MONO,
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
        sx={{ flexShrink: 0 }}
      >
        {t`Copy`}
      </Button>
    </Box>
  )
}

function Failure({ game }: { game: string }) {
  const { t } = useLingui()
  const theme = useTheme()
  const info = useProfiles((s) => s.game)
  const failure = useLaunch((s) => s.failure)
  const dismiss = useLaunch((s) => s.dismissFailure)
  const start = useLaunch((s) => s.start)
  const cause = failure?.cause
  const mod = useMods((s) => s.mods.find((m) => m.key === cause?.modKey))
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  // The dialog unmounts with `failure`, so nothing fades out with stale text.
  if (!failure) {
    return null
  }
  const name = info?.name ?? ''
  const nexusID = profile && mod ? nexusIdOf(profile, mod) : 0
  const showLine = failure.hint === Hint.HintLaunchOptions && info?.installDir
  const showFlatpak = failure.hint === Hint.HintFlatpakFS
  return (
    <Dialog
      open={true}
      onClose={dismiss}
      fullWidth={true}
      maxWidth="sm"
      slotProps={{ paper: { sx: { bgcolor: 'var(--mortar-panel-solid)' } } }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
        <CircleAlert size={22} color={theme.palette.error.light} aria-hidden={true} />
        {t`${name} did not start`}
      </DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1.75 }}>
        <DialogContentText sx={{ whiteSpace: 'pre-wrap', userSelect: 'text' }}>
          {failure.body || t`The game exited before it started. The console has its output.`}
        </DialogContentText>
        {cause ? (
          <DialogContentText>
            <strong>{t`Caused by ${cause.modName}`}</strong>
            <br />
            {cause.detail}
          </DialogContentText>
        ) : null}
        {showLine ? <LaunchLine line={launchLine(info.installDir)} /> : null}
        {showFlatpak ? <FlatpakGrant /> : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={dismiss}>{t`Close`}</Button>
        <Button
          onClick={() => {
            dismiss()
            useTab.getState().setTab('console')
          }}
        >
          {t`Open console`}
        </Button>
        {mod ? <SwitchOffButton mod={mod} /> : null}
        {nexusID > 0 ? (
          <Button
            onClick={() =>
              openPage(`https://www.nexusmods.com/stardewvalley/mods/${nexusID}`).catch(
                reportUnexpected,
              )
            }
          >
            {t`Open on Nexus`}
          </Button>
        ) : null}
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
    <ConfirmDialog
      open={ask !== null}
      title={t`Steam was not found`}
      body={
        ask?.profile === ''
          ? t`Mortar can start Stardew Valley directly instead. The Steam overlay and Steam's playtime tracking will not work while you play this way.`
          : t`Mortar can start SMAPI directly instead. The Steam overlay and Steam's playtime tracking will not work while you play this way.`
      }
      confirmLabel={t`Launch without Steam`}
      onCancel={() => answer(false)}
      onConfirm={() => {
        answer(true).catch(reportUnexpected)
      }}
    />
  )
}

export function LaunchLayer({ game }: { game: string }) {
  const refresh = useLaunch((s) => s.refresh)
  useEffect(() => {
    if (game) {
      refresh(game)
    }
  }, [game, refresh])
  return (
    <>
      <Overlay game={game} />
      <Failure game={game} />
      <DirectDialog />
      <CrashDialog />
      <UpdateWarnDialog />
      <SaveWarnDialog />
      <PrePlayDialog />
      <KnownGoodOffer />
      <VanillaPlayDialogs />
    </>
  )
}
