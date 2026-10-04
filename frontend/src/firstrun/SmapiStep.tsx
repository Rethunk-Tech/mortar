import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Typography } from '@mui/material'
import { alpha, useTheme } from '@mui/material/styles'

const STATUS_FILL = 0.14

import { System } from '@wailsio/runtime'
import { Check, Clock, Copy, Ellipsis, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import {
  LaunchOptions,
  SetLaunchOption,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { useLoader } from '../loader/store.ts'
import type { GameId } from '../nav/store.ts'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { MONO } from '../theme/theme.ts'
import { errorMessage, reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { launchLine, launchOptionsSet } from './logic.ts'
import { Panel } from './Panel.tsx'

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
  gameDir,
  options,
  recheck,
  onContinue,
}: {
  game: GameId
  gameDir: string
  options: string
  recheck: () => void
  onContinue: () => void
}) {
  const { t } = useLingui()
  const theme = useTheme()
  const line = launchLine(gameDir)
  const set = launchOptionsSet(options)
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
        {t`On Windows, Steam starts the game without SMAPI unless you tell it otherwise. With Steam closed, Mortar can set it for you; or in Steam, right-click Stardew Valley, choose Properties, and paste this line into Launch Options:`}
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
          sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Copy`}
        </Button>
      </Box>
      <Box>
        <Button
          variant="outlined"
          disabled={set || writing}
          onClick={write}
          sx={{ whiteSpace: 'nowrap' }}
        >
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
          borderColor: set ? 'success.main' : 'warning.main',
          bgcolor: set
            ? alpha(theme.palette.success.main, STATUS_FILL)
            : alpha(theme.palette.warning.main, STATUS_FILL),
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
          sx={{ height: 44, whiteSpace: 'nowrap' }}
        >
          {t`Check again`}
        </Button>
        <Button variant="contained" onClick={onContinue} sx={{ height: 44, whiteSpace: 'nowrap' }}>
          {t`Continue`}
        </Button>
      </Box>
    </>
  )
}

function CheckFailed({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useLingui()
  return (
    <Panel width={680}>
      <Typography role="alert" sx={{ fontSize: 14, color: 'error.light' }}>
        {message}
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
  error: string
  ready: boolean
  onRetry: () => void
}) {
  const { t } = useLingui()
  if (error !== '') {
    return <CheckFailed message={error} onRetry={onRetry} />
  }
  if (!ready) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  return null
}

export function SmapiStep({
  game,
  gameDir,
  onDone,
}: {
  game: GameId
  gameDir: string
  onDone: () => void
}) {
  const { t } = useLingui()
  const ok = useTheme().palette.success.main
  const windows = System.IsWindows()
  const status = useLoader((s) => s.status)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const check = useLoader((s) => s.check)
  const error = useLoader((s) => s.error)
  const errorDetail = useLoader((s) => s.errorDetail)
  const install = useLoader((s) => s.install)
  const pending = useLoader((s) => s.pending)
  const [options, setOptions] = useState('')
  const [checked, setChecked] = useState(false)
  const [checkError, setCheckError] = useState('')
  const entered = useRef(false)
  const started = useRef(false)
  const checkGen = useRef(0)

  const readOptions = useCallback(
    // Steam's config is unreadable until the user has run Steam once, which reads as "not set yet".
    () =>
      windows ? LaunchOptions(game).then(setOptions, () => setOptions('')) : Promise.resolve(),
    [windows, game],
  )
  const runCheck = useCallback(() => {
    checkGen.current += 1
    const token = checkGen.current
    setChecked(false)
    setCheckError('')
    Promise.all([check(game), readOptions()])
      .then(() => {
        if (token !== checkGen.current) {
          return
        }
        setChecked(true)
        setCheckError('')
      })
      .catch((e: unknown) => {
        if (token !== checkGen.current) {
          return
        }
        setCheckError(errorMessage(e))
      })
  }, [check, readOptions, game])
  useEffect(runCheck, [runCheck])

  const smapiReady = status?.installed === true && !status.broken
  const launchReady = !windows || launchOptionsSet(options)
  useEffect(() => {
    if (checked && !entered.current) {
      entered.current = true
      if (smapiReady && launchReady) {
        onDone()
      }
    }
  }, [checked, smapiReady, launchReady, onDone])

  // The install starts by itself; a failure waits for Retry instead of looping.
  useEffect(() => {
    if (checked && !smapiReady && !installing && !started.current) {
      started.current = true
      install(game)
    }
  }, [checked, smapiReady, installing, install, game])

  const waiting = <CheckGate error={checkError} ready={checked} onRetry={runCheck} />
  if (waiting) {
    return waiting
  }
  if (smapiReady && !installing) {
    const showLaunch = windows && !launchReady
    return (
      <Panel width={720}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, fontSize: 15 }}>
          <Check size={18} color={ok} />
          {t`SMAPI ${status.version} installed`}
        </Box>
        {showLaunch ? (
          <LaunchLine
            game={game}
            gameDir={gameDir}
            options={options}
            recheck={() => readOptions().catch(reportUnexpected)}
            onContinue={onDone}
          />
        ) : (
          <>
            <InstallLog steps={steps} installing={false} />
            <Button
              variant="contained"
              onClick={onDone}
              sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
            >
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
        {latest ? t`Installing SMAPI ${latest}` : t`Installing SMAPI`}
      </Typography>
      <Typography sx={{ fontSize: 15, lineHeight: 1.5 }}>
        {windows
          ? t`SMAPI is the loader every Stardew mod needs. It goes into the game folder.`
          : t`SMAPI is the loader every Stardew mod needs. It goes into the game folder, and Steam keeps launching the game as usual.`}
      </Typography>
      {installing ? (
        <>
          <LinearProgress variant="determinate" value={(steps.length / ORDER.length) * PERCENT} />
          <InstallLog steps={steps} installing={true} />
        </>
      ) : null}
      {error && !installing ? (
        <>
          <Typography role="alert" sx={{ fontSize: 14, color: 'error.light' }}>
            {error}
          </Typography>
          {errorDetail && errorDetail !== error ? (
            <Typography
              sx={{
                fontFamily: MONO,
                fontSize: 12,
                opacity: 0.8,
                wordBreak: 'break-word',
                userSelect: 'text',
              }}
            >
              {errorDetail}
            </Typography>
          ) : null}
          <Box
            sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 1.25 }}
          >
            <Button
              variant="outlined"
              color="inherit"
              disabled={pending}
              onClick={onDone}
              sx={{ height: 46, whiteSpace: 'nowrap' }}
            >
              {t`Skip for now`}
            </Button>
            <Button
              variant="contained"
              startIcon={<RefreshCw size={16} />}
              disabled={pending}
              onClick={() => install(game)}
              sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
            >
              {t`Retry`}
            </Button>
          </Box>
        </>
      ) : null}
    </Panel>
  )
}
