import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
  Typography,
} from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import {
  Status,
  Stop,
  SwitchOff,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/bisect/service.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const pollDelayMs = 500

interface ModResult {
  key: string
  uniqueId: string
  name: string
}

interface BisectStatus {
  state: string
  step: number
  total: number
  modsLeft: number
  result?: {
    mods: ModResult[]
  }
  error?: string
}

interface Props {
  game: string
  profile: string
  jobID: string | null
  onClose: () => void
}

export function BisectDialog({ game, profile, jobID, onClose }: Props) {
  const { t } = useLingui()
  const [status, setStatus] = useState<BisectStatus | null>(null)
  const [stopping, setStopping] = useState(false)

  useEffect(() => {
    if (!jobID) {
      setStatus(null)
      return
    }
    let active = true
    let timer: ReturnType<typeof globalThis.setTimeout> | undefined
    const poll = async () => {
      try {
        const next = (await Status(jobID)) as BisectStatus
        if (!active) {
          return
        }
        setStatus(next)
        if (next.state === 'starting' || next.state === 'running') {
          timer = globalThis.setTimeout(poll, pollDelayMs)
        }
      } catch (error) {
        if (active) {
          setStatus({
            state: 'failed',
            step: 0,
            total: 0,
            modsLeft: 0,
            error: errorDetails(error),
          })
        }
      }
    }
    poll().catch(() => undefined)
    return () => {
      active = false
      if (timer !== undefined) {
        globalThis.clearTimeout(timer)
      }
    }
  }, [jobID])

  const stop = async () => {
    if (!jobID) {
      onClose()
      return
    }
    setStopping(true)
    try {
      await Stop(jobID)
    } catch (error) {
      reportUnexpected(error)
    } finally {
      setStopping(false)
    }
  }

  const switchOff = async () => {
    const mods = status?.result?.mods ?? []
    try {
      await Promise.all(mods.map((mod) => SwitchOff(game, profile, mod.key, mod.uniqueId)))
      useProfiles.getState().open(profile)
      const names = mods.map((mod) => mod.name).join(' + ')
      useToasts.getState().push({
        kind: 'success',
        title: t`Switched off ${names}`,
        action: {
          label: t`Undo`,
          run: () =>
            Promise.all(
              mods.map((mod) => {
                const listed = useMods.getState().mods.find((m) => m.key === mod.key)
                return listed ? useMods.getState().setEnabled(listed, true) : Promise.resolve()
              }),
            ),
        },
      })
      onClose()
    } catch (error) {
      reportUnexpected(error)
    }
  }

  const done = status?.state === 'done'
  const stopped = status?.state === 'stopped'
  const failed = status?.state === 'failed'
  const names = status?.result?.mods.map((mod) => mod.name).join(' + ') ?? ''
  let content: ReactNode
  if (status === null || status.state === 'starting') {
    content = <Typography>{t`Preparing a temporary copy of this profile…`}</Typography>
  } else if (done) {
    content = (
      <>
        <Typography sx={{ fontWeight: 700 }}>{t`${names} seems to cause the crash`}</Typography>
        <Typography color="text.secondary">{t`The original profile was not changed.`}</Typography>
      </>
    )
  } else if (stopped) {
    content = <Typography>{t`Crash finding was stopped.`}</Typography>
  } else if (failed) {
    content = <Typography color="error">{status.error || t`Crash finding failed.`}</Typography>
  } else {
    content = (
      <>
        <Typography>
          {t`Step ${status.step} of ~${status.total}, ${plural(status.modsLeft, { one: '# mod left', other: '# mods left' })}`}
        </Typography>
        <LinearProgress />
      </>
    )
  }

  let actions: ReactNode
  if (done) {
    actions = (
      <>
        <Button onClick={onClose}>{t`Close`}</Button>
        <Button
          onClick={() => {
            useProfiles.getState().open(profile)
            onClose()
          }}
        >{t`Open profile`}</Button>
        <Button variant="contained" onClick={switchOff}>{t`Switch off`}</Button>
      </>
    )
  } else if (stopped || failed) {
    actions = <Button onClick={onClose}>{t`Close`}</Button>
  } else {
    actions = (
      <Button onClick={stop} disabled={stopping}>
        {stopping ? t`Stopping…` : t`Stop`}
      </Button>
    )
  }

  return (
    <Dialog
      open={jobID !== null}
      onClose={done || stopped || failed ? onClose : undefined}
      transitionDuration={0}
    >
      <DialogTitle sx={{ fontSize: 22, fontWeight: 700 }}>
        {t`Finding the mod causing the crash`}
      </DialogTitle>
      <DialogContent sx={{ minWidth: 440, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
        {content}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2.5 }}>{actions}</DialogActions>
    </Dialog>
  )
}
