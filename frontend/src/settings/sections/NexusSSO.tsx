import { useLingui } from '@lingui/react/macro'
import { Box, Button, CircularProgress } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { LogIn } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  CancelSSO,
  SSOAvailable,
  StartSSO,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { reportUnexpected, toastError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'

type Stage = 'idle' | 'connected' | 'waiting-browser'

export function NexusSSO() {
  const { t } = useLingui()
  const [available, setAvailable] = useState(false)
  const [stage, setStage] = useState<Stage>('idle')
  useEffect(() => {
    SSOAvailable().then(setAvailable).catch(reportUnexpected)
  }, [])
  useEffect(
    () =>
      Events.On('nexus:sso', (event) => {
        const { state, error, cancelled } = event.data
        if (state === 'connected' || state === 'waiting-browser') {
          setStage(state)
          return
        }
        setStage('idle')
        if (state === 'done') {
          useToasts.getState().push({ kind: 'success', title: t`Signed in to Nexus Mods` })
        } else if (state === 'failed' && !cancelled) {
          toastError(t`Could not sign in with Nexus Mods`, error)
        }
      }),
    [t],
  )
  if (!available) {
    return null
  }
  const start = () => {
    setStage('connected')
    StartSSO().catch(() => setStage('idle'))
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, pb: 1.5 }}>
      <Button
        variant="contained"
        startIcon={stage === 'idle' ? <LogIn size={16} /> : <CircularProgress size={16} />}
        disabled={stage !== 'idle'}
        onClick={start}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {t`Sign in to Nexus Mods`}
      </Button>
      {stage === 'connected' ? <Box sx={{ fontSize: 14 }}>{t`Connecting…`}</Box> : null}
      {stage === 'waiting-browser' ? (
        <>
          <Box sx={{ fontSize: 14 }}>{t`Approve Mortar in your browser…`}</Box>
          <Button
            variant="outlined"
            color="inherit"
            onClick={() => CancelSSO().catch(reportUnexpected)}
          >
            {t`Cancel`}
          </Button>
        </>
      ) : null}
    </Box>
  )
}
