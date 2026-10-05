import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Typography } from '@mui/material'
import { useTheme } from '@mui/material/styles'

import { System } from '@wailsio/runtime'
import { Check, Clock, Copy, Ellipsis, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import {
  LaunchOptions,
  LaunchOptionsStartLoader,
  SetLaunchOption,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { useGameName } from '../games/info.ts'
import { useLoader } from '../loader/store.ts'
import type { GameId } from '../nav/store.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { MONO } from '../theme/theme.ts'
import { type InlineError, inlineError, reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { Panel } from './Panel.tsx'
import { useLaunchLine } from './useLaunchLine.ts'

const PERCENT = 100
const ORDER = ['downloaded', 'files', 'launcher', 'bundled'] as const

function InstallLog({ steps, installing }: { steps: string[]; installing: boolean }) {
  const { t } = useLingui()
  const labels: Record<(typeof ORDER)[number], string> = {
    downloaded: t`Downloaded`,
    files: t`Files added`,
    launcher: t`Launcher replaced`,
    bundled: t`Bundled mods added`,
  }
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: '4px',
        p: '12px 14px',
        bgcolor: 'var(--mortar-overlay-45)',
        borderRadius: '6px',
        fontFamily: MONO,
        fontSize: 13,
      }}
    >
      {ORDER.map((step, i) => {
        const done = steps.includes(step)
        if (!(done || (installing && i === steps.length))) {
          return null
        }
        return (
          <Box
            key={step}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              color: done ? 'success.light' : 'var(--mortar-ink)',
            }}
          >
            {done ? <Check size={14} /> : <Ellipsis size={14} />}
            {labels[step]}
          </Box>
        )
      })}
    </Box>
  )
}

function LaunchLine({
  game,
  loader,
  gameDir,
  set,
  recheck,
  onContinue,
}: {
  game: GameId
  loader: string
  gameDir: string
  set: boolean
  recheck: () => void
  onContinue: () => void
}) {
  const { t } = useLingui()
  const theme = useTheme()
  const gameName = useGameName(game)
  const line = useLaunchLine(game, gameDir)
  const copy = () => {
    navigator.clipboard
      .writeText(line)
      .then(() => useToasts.getState().push({ kind: 'success', title: t`Launch options copied` }))
      .catch(reportUnexpected)
  }
  const [writing, setWriting] = useState(false)
  const write = () => {
    setWriting(true)
    SetLaunchOption(game)
      .then(() => {
        useToasts.getState().push({ kind: 'success', title: t`Launch options set in Steam` })
        recheck()
      })
      .catch(reportError(t`Could not set it in Steam`))
      .finally(() => setWriting(false))
  }
  return (
    <>
      <Typography sx={{ fontSize: 22, fontWeight: 700 }}>{t`One step in Steam`}</Typography>
      <Typography sx={{ fontSize: 15, lineHeight: 1.5 }}>
        {t`On Windows, Steam starts the game without ${loader} unless told to. With Steam closed, Mortar can set that up; or in Steam, right-click ${gameName}, choose Properties and paste this line into Launch Options:`}
      </Typography>
      <Box sx={{ display: 'flex', gap: 1 }}>
        <Box
          sx={{
            flex: 1,
            minWidth: 0,
            display: 'flex',
            alignItems: 'center',
            minHeight: 44,
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
          color="inherit"
          startIcon={<Copy size={16} />}
          onClick={copy}
          sx={{ flexShrink: 0 }}
        >
          {t`Copy`}
        </Button>
      </Box>
      <Box>
        <Button variant="outlined" disabled={set || writing} onClick={write}>
          {t`Set it in Steam for me`}
        </Button>
      </Box>
      <Box
        role="status"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.25,
          px: 1.5,
          py: 1.25,
          fontSize: 14,
          borderRadius: '6px',
          border: '1px solid',
          borderColor: calloutLine(set ? 'success' : 'warning'),
          bgcolor: calloutFill(set ? 'success' : 'warning'),
        }}
      >
        {set ? (
          <Check size={16} color={theme.palette.success.main} />
        ) : (
          <Clock size={16} color={theme.palette.warning.main} />
        )}
        {set
          ? t`Mortar found it in Steam's settings.`
          : t`Mortar checks Steam's settings for it: not found yet.`}
      </Box>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 1.25 }}>
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<RefreshCw size={16} />}
          onClick={recheck}
          size="large"
        >
          {t`Check again`}
        </Button>
        <Button variant="contained" onClick={onContinue} size="large">
          {t`Continue`}
        </Button>
      </Box>
    </>
  )
}

function CheckFailed({ error, onRetry }: { error: InlineError; onRetry: () => void }) {
  const { t } = useLingui()
  return (
    <Panel width={680}>
      <Typography role="alert" title={error.details} sx={{ fontSize: 14, color: 'error.light' }}>
        {error.message}
      </Typography>
      <Button variant="contained" onClick={onRetry}>
        {t`Retry`}
      </Button>
    </Panel>
  )
}

function CheckGate({
  error,
  ready,
  onRetry,
}: {
  error: InlineError | null
  ready: boolean
  onRetry: () => void
}) {
  const { t } = useLingui()
  if (error) {
    return <CheckFailed error={error} onRetry={onRetry} />
  }
  if (!ready) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  return null
}

function InstallFailed({
  error,
  detail,
  pending,
  onSkip,
  onRetry,
}: {
  error: string
  detail: string
  pending: boolean
  onSkip: () => void
  onRetry: () => void
}) {
  const { t } = useLingui()
  return (
    <>
      <Typography role="alert" sx={{ fontSize: 14, color: 'error.light' }}>
        {error}
      </Typography>
      {detail && detail !== error ? (
        <Typography
          sx={{
            fontFamily: MONO,
            fontSize: 12,
            opacity: 0.8,
            wordBreak: 'break-word',
            userSelect: 'text',
          }}
        >
          {detail}
        </Typography>
      ) : null}
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 1.25 }}>
        <Button variant="outlined" color="inherit" disabled={pending} onClick={onSkip} size="large">
          {t`Skip for now`}
        </Button>
        <Button
          variant="contained"
          startIcon={<RefreshCw size={16} />}
          disabled={pending}
          onClick={onRetry}
          size="large"
        >
          {t`Retry`}
        </Button>
      </Box>
    </>
  )
}

export function LoaderStep({
  game,
  loader,
  gameDir,
  onDone,
}: {
  game: GameId
  loader: string
  gameDir: string
  onDone: () => void
}) {
  const { t } = useLingui()
  const ok = useTheme().palette.success.main
  const gameName = useGameName(game)
  const windows = System.IsWindows()
  const status = useLoader((s) => s.status)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const check = useLoader((s) => s.check)
  const error = useLoader((s) => s.error)
  const errorDetail = useLoader((s) => s.errorDetail)
  const install = useLoader((s) => s.install)
  const pending = useLoader((s) => s.pending)
  const [launchSet, setLaunchSet] = useState(false)
  const [checked, setChecked] = useState(false)
  const [checkError, setCheckError] = useState<InlineError | null>(null)
  const entered = useRef(false)
  const started = useRef(false)
  const checkGen = useRef(0)

  const readOptions = useCallback(
    // Steam's config is unreadable until the user has run Steam once, which reads as "not set yet".
    () =>
      windows
        ? LaunchOptions(game)
            .then((options) => LaunchOptionsStartLoader(game, options))
            .then(setLaunchSet, () => setLaunchSet(false))
        : Promise.resolve(),
    [windows, game],
  )
  const runCheck = useCallback(() => {
    checkGen.current += 1
    const token = checkGen.current
    setChecked(false)
    setCheckError(null)
    Promise.all([check(game), readOptions()])
      .then(() => {
        if (token !== checkGen.current) {
          return
        }
        setChecked(true)
        setCheckError(null)
      })
      .catch((e: unknown) => {
        if (token !== checkGen.current) {
          return
        }
        setCheckError(inlineError(e))
      })
  }, [check, readOptions, game])
  useEffect(runCheck, [runCheck])

  const loaderReady = status?.installed === true && !status.broken
  const launchReady = !windows || launchSet
  useEffect(() => {
    if (checked && !entered.current) {
      entered.current = true
      if (loaderReady && launchReady) {
        onDone()
      }
    }
  }, [checked, loaderReady, launchReady, onDone])

  // The install starts by itself; a failure waits for Retry instead of looping.
  useEffect(() => {
    if (checked && !loaderReady && !installing && !started.current) {
      started.current = true
      install(game)
    }
  }, [checked, loaderReady, installing, install, game])

  if (checkError || !checked) {
    return <CheckGate error={checkError} ready={checked} onRetry={runCheck} />
  }
  if (loaderReady && !installing) {
    const showLaunch = windows && !launchReady
    return (
      <Panel width={720}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, fontSize: 15 }}>
          <Check size={18} color={ok} />
          {t`${loader} ${{ version: status.version }} installed`}
        </Box>
        {showLaunch ? (
          <LaunchLine
            game={game}
            loader={loader}
            gameDir={gameDir}
            set={launchSet}
            recheck={() => readOptions().catch(reportUnexpected)}
            onContinue={onDone}
          />
        ) : (
          <>
            <InstallLog steps={steps} installing={false} />
            <Button variant="contained" onClick={onDone} size="large">
              {t`Continue`}
            </Button>
          </>
        )}
      </Panel>
    )
  }
  const latest = status?.latest ?? ''
  return (
    <Panel width={680}>
      <Typography sx={{ fontSize: 22, fontWeight: 700 }}>
        {latest ? t`Installing ${loader} ${latest}` : t`Installing ${loader}`}
      </Typography>
      <Typography sx={{ fontSize: 15, lineHeight: 1.5 }}>
        {windows
          ? t`${loader} is the loader every ${gameName} mod needs. It goes into the game folder.`
          : t`${loader} is the loader every ${gameName} mod needs. It goes into the game folder, and Steam keeps launching the game as usual.`}
      </Typography>
      {installing ? (
        <>
          <LinearProgress variant="determinate" value={(steps.length / ORDER.length) * PERCENT} />
          <InstallLog steps={steps} installing={true} />
        </>
      ) : null}
      {error && !installing ? (
        <InstallFailed
          error={error}
          detail={errorDetail}
          pending={pending}
          onSkip={onDone}
          onRetry={() => install(game)}
        />
      ) : null}
    </Panel>
  )
}
