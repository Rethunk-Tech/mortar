import { useLingui } from '@lingui/react/macro'
import { Box, Button, CircularProgress } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { LogIn } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  CancelSSO,
  StartSSO,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { openWeb } from '../../shell/openWeb.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected, toastError } from '../../toasts/report.ts'

type Stage = 'idle' | 'connected' | 'waiting-browser'

export function NexusSSO() {
  const { t } = useLingui()
  const [stage, setStage] = useState<Stage>('idle')
  useEffect(
    () =>
      Events.On('nexus:sso', (event) => {
        const { state, error, cancelled, url } = event.data
        if (state === 'open-in-page') {
          if (url) {
            openWeb(url).catch(reportUnexpected)
          }
          return
        }
        if (state === 'connected' || state === 'waiting-browser') {
          setStage(state)
          return
        }
        setStage('idle')
        if (state === 'failed' && !cancelled) {
          toastError(t`Could not sign in with ${{ source: sourceLabel('nexus') }}`, error)
        }
      }),
    [t],
  )
  const start = () => {
    setStage('connected')
    StartSSO().catch(() => setStage('idle'))
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <Button
        variant="contained"
        size="small"
        startIcon={stage === 'idle' ? <LogIn size={16} /> : <CircularProgress size={16} />}
        disabled={stage !== 'idle'}
        onClick={start}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {t`Sign in to ${{ source: sourceLabel('nexus') }}`}
      </Button>
      {stage === 'connected' ? (
        <Box sx={{ fontSize: 'medium.fontSize' }}>{t`Connecting…`}</Box>
      ) : null}
      {stage === 'waiting-browser' ? (
        <>
          <Box sx={{ fontSize: 'medium.fontSize' }}>{t`Approve Mortar in your browser…`}</Box>
          <Button
            variant="outlined"
            size="small"
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
