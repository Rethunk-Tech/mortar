import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Typography } from '@mui/material'
import { System } from '@wailsio/runtime'
import { Check, Clock, Copy, Ellipsis, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { LaunchOptions } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { useLoader } from '../loader/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { launchLine, launchOptionsSet } from './logic.ts'
import { STARDEW } from './needed.ts'
import { Panel } from './Panel.tsx'

const MONO = '"IBM Plex Mono", monospace'
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
        bgcolor: 'rgba(0,0,0,0.45)',
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
              color: done ? 'success.light' : '#fff',
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
  gameDir,
  options,
  recheck,
  onContinue,
}: {
  gameDir: string
  options: string
  recheck: () => void
  onContinue: () => void
}) {
  const { t } = useLingui()
  const line = launchLine(gameDir)
  const set = launchOptionsSet(options)
  const copy = () => {
    navigator.clipboard
      .writeText(line)
      .then(() => useToasts.getState().push({ kind: 'success', title: t`Launch options copied` }))
      .catch(reportUnexpected)
  }
  return (
    <>
      <Typography sx={{ fontSize: 22, fontWeight: 700 }}>{t`One step in Steam`}</Typography>
      <Typography sx={{ fontSize: 15, lineHeight: 1.5 }}>
        {t`On Windows, Steam starts the game without SMAPI unless you tell it otherwise. In Steam, right-click Stardew Valley, choose Properties, and paste this line into Launch Options:`}
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
            bgcolor: 'rgba(0,0,0,0.45)',
            border: '1px solid rgba(255,255,255,0.15)',
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
          variant="contained"
          startIcon={<Copy size={16} />}
          onClick={copy}
          sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Copy`}
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
          bgcolor: set ? 'rgba(12,223,100,0.14)' : 'rgba(243,180,22,0.14)',
        }}
      >
        {set ? <Check size={16} color="#0CDF64" /> : <Clock size={16} color="#F3B416" />}
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

export function SmapiStep({ gameDir, onDone }: { gameDir: string; onDone: () => void }) {
  const { t } = useLingui()
  const windows = System.IsWindows()
  const status = useLoader((s) => s.status)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const check = useLoader((s) => s.check)
  const error = useLoader((s) => s.error)
  const install = useLoader((s) => s.install)
  const pending = useLoader((s) => s.pending)
  const [options, setOptions] = useState('')
  const [checked, setChecked] = useState(false)
  const entered = useRef(false)
  const started = useRef(false)

  const readOptions = useCallback(
    // Steam's config is unreadable until the user has run Steam once, which reads as "not set yet".
    () =>
      windows ? LaunchOptions(STARDEW).then(setOptions, () => setOptions('')) : Promise.resolve(),
    [windows],
  )
  useEffect(() => {
    Promise.all([check(STARDEW), readOptions()])
      .then(() => setChecked(true))
      .catch(reportUnexpected)
  }, [check, readOptions])

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
      install(STARDEW)
    }
  }, [checked, smapiReady, installing, install])

  if (!checked) {
    return null
  }
  if (smapiReady && !installing) {
    const showLaunch = windows && !launchReady
    return (
      <Panel width={720}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, fontSize: 15 }}>
          <Check size={18} color="#0CDF64" />
          {t`SMAPI ${status.version} installed`}
        </Box>
        {showLaunch ? (
          <LaunchLine
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
          <Button
            variant="contained"
            startIcon={<RefreshCw size={16} />}
            disabled={pending}
            onClick={() => install(STARDEW)}
            sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
          >
            {t`Retry`}
          </Button>
        </>
      ) : null}
    </Panel>
  )
}
