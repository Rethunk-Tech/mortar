import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Modal, Typography } from '@mui/material'
import { Puzzle } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { Level } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { Progress } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { CancelLaunch } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { OpenSteam } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import { LauncherLogo } from '../brand/launchers/LauncherLogo.tsx'
import { loaderLogo } from '../brand/loaders/loaderLogo.ts'
import { EPIC_GAMES, HEROIC, LUTRIS, STEAM } from '../brand/productNames.ts'
import { SwitchOffButton } from '../console/CrashDialog.tsx'
import { format } from '../console/filter.ts'
import { useConsole } from '../console/store.ts'
import { useTab } from '../game/tab.ts'
import { gameArt } from '../games/art.ts'
import { useGameInfo } from '../games/info.ts'
import { modsLabel } from '../i18n/counts.ts'
import { i18n } from '../i18n/index.ts'
import { useReasonText } from '../i18n/reasonText.ts'
import { useMods } from '../mods/store.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { controlsCutout } from '../shell/controlsCutout.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { LaunchStages, type StageMark } from './LaunchStages.tsx'
import { type Phase, useLaunchView } from './progress.ts'
import { launchStages, type StageId } from './stages.ts'
import { useLaunch } from './store.ts'

const LOG_LINES = 5
const LINE_HEIGHT = 21
const LOG_PAD = 12
const LOG_HEIGHT = LOG_LINES * LINE_HEIGHT + 2 * LOG_PAD
const ART = 60
const RUNNING_MS = 1200
// A launch screen whose marks never all show done (its progress was lost) still closes once the game runs.
const RUNNING_CAP_MS = 4000
const SECOND = 1000

const scrim = {
  position: 'fixed',
  inset: 0,
  zIndex: 1200,
  bgcolor: 'var(--mortar-overlay-80)',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  gap: space.pad,
  p: space.pad,
  clipPath: controlsCutout,
} as const

const launcherNames: Record<string, string> = {
  steam: STEAM,
  'flatpak-steam': STEAM,
  heroic: HEROIC,
  lutris: LUTRIS,
  epic: EPIC_GAMES,
}

// useNow is the time, refreshed every second while on.
function useNow(on: boolean): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (!on) {
      return
    }
    const timer = setInterval(() => setNow(Date.now()), SECOND)
    return () => clearInterval(timer)
  }, [on])
  return now
}

function clock(ms: number): string {
  const s = Math.max(0, Math.floor(ms / SECOND))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

// Art is the first of srcs that loads, else fallback.
function Art({ srcs, fallback }: { srcs: string[]; fallback: ReactNode }) {
  const [failed, setFailed] = useState(0)
  const src = srcs.filter(Boolean)[failed]
  if (src === undefined) {
    return <Box sx={{ display: 'grid', placeItems: 'center' }}>{fallback}</Box>
  }
  return (
    <Box
      component="img"
      key={src}
      src={src}
      alt=""
      onError={() => setFailed((n) => n + 1)}
      sx={{
        width: '100%',
        height: '100%',
        objectFit: 'cover',
        borderRadius: '10px',
        imageRendering: 'pixelated',
      }}
    />
  )
}

// The panel over the log: the slow-launch notice or the failure, so neither moves anything on the screen.
function Panel({ children, error }: { children: ReactNode; error?: boolean }) {
  return (
    <Box
      sx={{
        position: 'absolute',
        inset: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 1,
        px: 3,
        textAlign: 'center',
        bgcolor: 'var(--mortar-panel-solid)',
        border: '1px solid',
        borderColor: error ? 'error.main' : 'primary.dark',
      }}
    >
      {children}
    </Box>
  )
}

function gameText(p: Progress): string {
  const up = p.up >= p.copies || p.loaderStarted
  if (p.copies > 1) {
    return up
      ? plural({ count: p.copies }, { one: '# copy running', other: '# copies running' })
      : i18n._(msg`Starting copy ${{ copy: p.up + 1 }} of ${{ copies: p.copies }}`)
  }
  return up ? i18n._(msg`Running`) : i18n._(msg`Starting`)
}

function loaderText(p: Progress, failedMod: string): string {
  if (failedMod) {
    return i18n._(msg`${{ plugin: failedMod }} failed to load.`)
  }
  if (p.ready) {
    return p.loaded > 0
      ? plural({ count: p.loaded }, { one: '# mod loaded', other: '# mods loaded' })
      : i18n._(msg`Mods loaded`)
  }
  if (p.loaded > 0 && p.total > 0) {
    return i18n._(msg`Loading ${{ loaded: p.loaded }} of ${{ total: p.total }} mods`)
  }
  if (p.loaded > 0) {
    return plural({ count: p.loaded }, { one: 'Loaded # mod', other: 'Loaded # mods' })
  }
  return p.loaderStarted ? i18n._(msg`Loading mods`) : i18n._(msg`Waiting`)
}

// marksFor are the launch's marks: the launcher, the game and its loader, each with what it is doing.
function marksFor(
  p: Progress,
  failedMod: string,
  info: GameInfo | undefined,
): Record<StageId, StageMark> {
  const launcher = launcherNames[p.via] ?? ''
  return {
    via: {
      name: launcher,
      art: <LauncherLogo id={p.via} size={ART} />,
      text: p.handed || p.up > 0 ? i18n._(msg`Handed off`) : i18n._(msg`Asking ${launcher}`),
      share: null,
    },
    game: {
      name: info?.name ?? '',
      art: (
        <Art
          srcs={[info?.iconUrl ?? '', gameArt(info, ART)]}
          fallback={
            <Typography sx={{ fontSize: 28, fontWeight: 700 }}>
              {(info?.name ?? '?').slice(0, 1)}
            </Typography>
          }
        />
      ),
      text: gameText(p),
      share: null,
    },
    loader: {
      name: info?.loader ?? '',
      art: <Art srcs={[loaderLogo(p.loader)]} fallback={<Puzzle size={40} aria-hidden={true} />} />,
      text: loaderText(p, failedMod),
      share: p.tracks && p.total > 0 ? Math.min(1, p.loaded / p.total) : null,
    },
  }
}

function slowTitle(p: Progress, stage: StageId, info: GameInfo | undefined): string {
  const name = info?.name ?? ''
  const loaderName = info?.loader ?? ''
  if (stage === 'via') {
    return i18n._(msg`${{ launcher: launcherNames[p.via] ?? '' }} has not started ${name} yet`)
  }
  if (stage === 'game') {
    return i18n._(msg`${name} has not started yet`)
  }
  if (!p.loaderStarted) {
    return i18n._(msg`${name} is running, but ${loaderName} has not started`)
  }
  if (!p.mod) {
    return i18n._(msg`${loaderName} is still loading mods`)
  }
  return p.total > 0
    ? i18n._(
        msg`Still loading ${{ mod: p.mod }} (${{ loaded: p.loaded }} of ${{ total: p.total }})`,
      )
    : i18n._(msg`Still loading ${{ mod: p.mod }}`)
}

function slowBody(p: Progress, stage: StageId): string {
  if (stage === 'via') {
    return i18n._(
      msg`${{ launcher: launcherNames[p.via] ?? '' }} may be updating the game, or it is not signed in. Mortar keeps waiting.`,
    )
  }
  if (stage === 'game' || !p.loaderStarted) {
    return i18n._(msg`The game may still be loading its own files. Mortar keeps waiting.`)
  }
  return i18n._(msg`A long mod list can take a while on its first start. Mortar keeps waiting.`)
}

// The slow notice names the stage the launch is stuck on.
function SlowNotice({ p, stage, onKeep }: { p: Progress; stage: StageId; onKeep: () => void }) {
  const { t } = useLingui()
  const info = useGameInfo(p.game)
  const launcher = launcherNames[p.via] ?? ''
  const title = slowTitle(p, stage, info)
  const body = slowBody(p, stage)
  return (
    <Panel>
      <Typography sx={{ fontSize: 17, fontWeight: 600 }}>{title}</Typography>
      <Typography sx={{ color: 'text.secondary', fontSize: 14 }}>{body}</Typography>
      <Box sx={{ display: 'flex', gap: space.gap, mt: 0.5 }}>
        {stage === 'via' && launcherNames[p.via] === STEAM ? (
          <Button
            onClick={() => OpenSteam().catch(reportUnexpected)}
          >{t`Open ${{ name: launcher }}`}</Button>
        ) : null}
        <Button variant="contained" autoFocus={true} onClick={onKeep}>
          {t`Keep waiting`}
        </Button>
      </Box>
    </Panel>
  )
}

function FailurePanel({ game }: { game: string }) {
  const { t } = useLingui()
  const reasonText = useReasonText()
  const failure = useLaunch((s) => s.failure)
  const dismiss = useLaunch((s) => s.dismissFailure)
  const start = useLaunch((s) => s.start)
  const close = useLaunchView((s) => s.close)
  const cause = failure?.cause
  const mod = useMods((s) => s.mods.find((m) => m.key === cause?.modKey))
  const name = useGameInfo(game)?.name ?? ''
  if (!failure) {
    return null
  }
  const body = cause
    ? reasonText(cause.reason, cause.params, cause.detail)
    : failure.body || t`The game exited before it started. The console has its output.`
  return (
    <Panel error={true}>
      <Typography sx={{ fontSize: 17, fontWeight: 600 }}>
        {cause
          ? t`${{ name: cause.modName }} stopped the game from starting`
          : t`${name} did not start`}
      </Typography>
      <Typography
        sx={{ color: 'text.secondary', fontSize: 14, userSelect: 'text', maxWidth: '100%' }}
        noWrap={true}
        title={body}
      >
        {body}
      </Typography>
      <Box sx={{ display: 'flex', gap: space.gap, mt: 0.5 }}>
        {mod ? <SwitchOffButton mod={mod} /> : null}
        <Button
          variant="contained"
          autoFocus={true}
          onClick={() => {
            dismiss()
            close()
            start(game, failure.profile, false)
          }}
        >
          {t`Retry`}
        </Button>
      </Box>
    </Panel>
  )
}

// toneOf colours a log line: errors red, warnings amber, the newest line bright.
function toneOf(level: Level, newest: boolean): string {
  if (level === Level.Error || level === Level.Alert) {
    return 'error.light'
  }
  if (level === Level.Warn) {
    return 'warning.light'
  }
  return newest ? 'var(--mortar-ink)' : 'inherit'
}

function Log() {
  const entries = useConsole((s) => s.entries)
  // Lines for each mod loading are counted by the loader's mark instead.
  const shown = entries.filter((e) => !e.load).slice(-LOG_LINES)
  return (
    <Box
      role="log"
      sx={{
        height: LOG_HEIGHT,
        boxSizing: 'border-box',
        overflow: 'hidden',
        p: `${LOG_PAD}px`,
        bgcolor: 'var(--mortar-overlay-55)',
        fontFamily: MONO,
        fontSize: 13,
        lineHeight: `${LINE_HEIGHT}px`,
        color: 'text.secondary',
        userSelect: 'text',
      }}
    >
      {shown.map((line, i) => {
        const text = format(line)
        const tone = toneOf(line.level, i === shown.length - 1)
        return (
          <Typography
            key={line.seq}
            noWrap={true}
            title={text}
            sx={{ font: 'inherit', color: tone }}
          >
            {text}
          </Typography>
        )
      })}
    </Box>
  )
}

function screenTitle(phase: Phase, name: string): string {
  if (phase === 'failed') {
    return i18n._(msg`${name} did not start`)
  }
  return phase === 'running' ? i18n._(msg`${name} is running`) : i18n._(msg`Launching ${name}`)
}

// Buttons under the log: Cancel (Close once failed), the console, and Hide; gone while the game shows as running.
function Buttons({ game, phase, focus }: { game: string; phase: Phase; focus: boolean }) {
  const { t } = useLingui()
  const hide = useLaunch((s) => s.hide)
  const dismissFailure = useLaunch((s) => s.dismissFailure)
  const close = useLaunchView((s) => s.close)
  const setTab = useTab((s) => s.setTab)
  const failed = phase === 'failed'
  const closeFailure = () => {
    dismissFailure()
    close()
  }
  const cancel = () => {
    CancelLaunch(game).catch(reportError(t`Could not cancel the launch`))
  }
  return (
    <Box
      sx={{
        display: 'flex',
        gap: space.gap,
        visibility: phase === 'running' ? 'hidden' : 'visible',
      }}
    >
      <Button onClick={failed ? closeFailure : cancel}>{failed ? t`Close` : t`Cancel`}</Button>
      <Button
        variant="outlined"
        autoFocus={focus}
        onClick={() => {
          setTab('console')
          if (failed) {
            closeFailure()
          } else {
            hide()
          }
        }}
      >
        {t`Open console`}
      </Button>
      {failed ? null : (
        <Button variant="outlined" onClick={hide}>
          {t`Hide`}
        </Button>
      )}
    </Box>
  )
}

/**
 * LaunchScreen covers the game page while a launch runs: the launcher, the game and its loader as a row of marks that
 * fill in as each comes up, the launch's own log lines, and a notice when it is slow. It stays up until the loader
 * says the mods have loaded, says the game is running for a moment, then closes; a failure stays on it.
 */
export function LaunchScreen({ game }: { game: string }) {
  const { t } = useLingui()
  const phase = useLaunchView((s) => s.phase)
  const viewGame = useLaunchView((s) => s.game)
  const progress = useLaunchView((s) => s.progress)
  const refresh = useLaunchView((s) => s.refresh)
  const close = useLaunchView((s) => s.close)
  const status = useLaunch((s) => s.status)
  const hidden = useLaunch((s) => s.hidden)
  const failure = useLaunch((s) => s.failure)
  const info = useGameInfo(game)
  const profile = useProfiles((s) =>
    s.profiles.find((p) => p.id === (progress?.profile ?? status?.profile)),
  )
  const open = phase !== 'closed' && viewGame === game && !hidden
  const failed = phase === 'failed'
  const now = useNow(open && !failed && phase !== 'running')
  const [settled, setSettled] = useState(false)
  const [keptWaiting, setKeptWaiting] = useState(0)
  const onSettled = useCallback(() => setSettled(true), [])

  useEffect(() => {
    if (open && progress === null) {
      refresh(game)
    }
  }, [open, progress, game, refresh])
  useEffect(() => {
    if (phase === 'launching') {
      setSettled(false)
    }
  }, [phase])
  useEffect(() => {
    if (phase !== 'running') {
      return
    }
    const timer = setTimeout(close, settled ? RUNNING_MS : RUNNING_CAP_MS)
    return () => clearTimeout(timer)
  }, [phase, settled, close])

  if (!open || progress === null) {
    return null
  }
  const name = info?.name ?? ''
  const stages = launchStages(progress, failed)
  const stuck = stages.find((s) => s.state === 'active')
  const slowAt = status?.slowAt ?? 0
  const since = status?.since ?? 0
  const slow =
    !failed &&
    phase !== 'running' &&
    stuck !== undefined &&
    slowAt > 0 &&
    now >= slowAt &&
    keptWaiting !== slowAt
  const failedMod = failed ? (failure?.cause?.modName ?? '') : ''
  const marks = marksFor(progress, failedMod, info)
  const broken = stages.find((s) => s.state === 'failed')
  if (broken && !(broken.id === 'loader' && failedMod)) {
    marks[broken.id] = { ...marks[broken.id], text: t`Did not start` }
  }
  const title = screenTitle(phase, name)
  const sub =
    progress.loader === ''
      ? t`Without mods`
      : `${profile?.name ?? ''} · ${modsLabel(profile ? userModCount(profile) : 0)}`
  const usual =
    progress.usualMs > 0
      ? t`Usually starts in about ${{ seconds: Math.round(progress.usualMs / SECOND) }} s`
      : ''
  const elapsed = since > 0 && !failed && phase !== 'running' ? clock(now - since) : ''
  return (
    <Modal
      open={true}
      onClose={useLaunch.getState().hide}
      hideBackdrop={true}
      sx={{ zIndex: scrim.zIndex, clipPath: controlsCutout }}
    >
      <Box role="dialog" aria-label={title} sx={scrim}>
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
        <Typography sx={{ fontSize: 30, fontWeight: 700 }}>{title}</Typography>
        <Typography sx={{ fontSize: 16 }}>{sub}</Typography>
        <Typography sx={{ fontSize: 13.5, color: 'text.secondary', minHeight: 19 }}>
          {[usual, elapsed].filter(Boolean).join(' · ')}
        </Typography>
        <LaunchStages stages={stages} marks={marks} onSettled={onSettled} />
        <Box sx={{ position: 'relative', width: '100%', maxWidth: 720 }}>
          <Log />
          {failed ? <FailurePanel game={game} /> : null}
          {slow && stuck ? (
            <SlowNotice p={progress} stage={stuck.id} onKeep={() => setKeptWaiting(slowAt)} />
          ) : null}
        </Box>
        <Buttons game={game} phase={phase} focus={!(slow || failed)} />
      </Box>
    </Modal>
  )
}
