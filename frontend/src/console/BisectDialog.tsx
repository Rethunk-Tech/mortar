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
import { useEffect, useState } from 'react'
import * as Bisect from '../../bindings/github.com/Rethunk-AI/mortar/internal/bisect/service.js'
import { useProfiles } from '../profiles/store.ts'

type ModResult = {
  key: string
  uniqueId: string
  name: string
}

type BisectStatus = {
  state: string
  step: number
  total: number
  modsLeft: number
  result?: {
    mods: ModResult[]
  }
  error?: string
}

type Props = {
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
    let timer: number | undefined
    const poll = async () => {
      try {
        const next = (await Bisect.Status(jobID)) as BisectStatus
        if (!active) {
          return
        }
        setStatus(next)
        if (next.state === 'starting' || next.state === 'running') {
          timer = window.setTimeout(() => void poll(), 500)
        }
      } catch (error) {
        if (active) {
          setStatus({
            state: 'failed',
            step: 0,
            total: 0,
            modsLeft: 0,
            error: error instanceof Error ? error.message : String(error),
          })
        }
      }
    }
    void poll()
    return () => {
      active = false
      if (timer !== undefined) {
        window.clearTimeout(timer)
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
      await Bisect.Stop(jobID)
    } finally {
      setStopping(false)
    }
  }

  const switchOff = async () => {
    const mods = status?.result?.mods ?? []
    await Promise.all(mods.map((mod) => Bisect.SwitchOff(game, profile, mod.key, mod.uniqueId)))
    useProfiles.getState().open(profile)
    onClose()
  }

  const done = status?.state === 'done'
  const stopped = status?.state === 'stopped'
  const failed = status?.state === 'failed'
  const names = status?.result?.mods.map((mod) => mod.name).join(' + ') ?? ''
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
        {status === null || status.state === 'starting' ? (
          <Typography>{t`Preparing a temporary copy of this profile…`}</Typography>
        ) : done ? (
          <>
            <Typography sx={{ fontWeight: 700 }}>{t`${names} seems to cause the crash`}</Typography>
            <Typography color="text.secondary">
              {t`The original profile was not changed.`}
            </Typography>
          </>
        ) : stopped ? (
          <Typography>{t`Crash finding was stopped.`}</Typography>
        ) : failed ? (
          <Typography color="error">{status.error || t`Crash finding failed.`}</Typography>
        ) : (
          <>
            <Typography>
              {t`Step ${status.step} of ~${status.total}, ${status.modsLeft} mods left`}
            </Typography>
            <LinearProgress />
          </>
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2.5 }}>
        {done ? (
          <>
            <Button onClick={onClose}>{t`Close`}</Button>
            <Button
              onClick={() => {
                useProfiles.getState().open(profile)
                onClose()
              }}
            >{t`Open page`}</Button>
            <Button variant="contained" onClick={() => void switchOff()}>{t`Switch off`}</Button>
          </>
        ) : stopped || failed ? (
          <Button onClick={onClose}>{t`Close`}</Button>
        ) : (
          <Button onClick={() => void stop()} disabled={stopping}>
            {stopping ? t`Stopping…` : t`Stop`}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}
