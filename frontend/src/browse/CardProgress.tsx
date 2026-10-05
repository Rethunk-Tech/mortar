import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, LinearProgress, Typography } from '@mui/material'
import { Check, RotateCw } from 'lucide-react'
import { Retry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { useQueue } from '../queue/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { CardState } from './cardState.ts'

const BAR_PX = 96

// The card's Add button once its mod is on its way: where it stands, and a retry when it failed.
function CardProgress({ state }: { state: Exclude<CardState, { kind: 'idle' }> }) {
  const { t } = useLingui()
  switch (state.kind) {
    case 'failed':
      return (
        <Button
          size="small"
          color="error"
          variant="outlined"
          startIcon={<RotateCw size={14} />}
          onClick={() => Retry(state.itemId).catch(reportUnexpected)}
        >
          {t`Failed, retry`}
        </Button>
      )
    case 'done':
      return <Chip size="small" color="primary" icon={<Check size={14} />} label={t`Installed`} />
    case 'attention':
      return (
        <Button size="small" variant="outlined" onClick={() => useQueue.getState().setOpen(true)}>
          {t`Needs your input`}
        </Button>
      )
    case 'downloading':
      return (
        <Box sx={{ width: BAR_PX }}>
          <Typography sx={{ fontSize: 12 }}>{t`Downloading ${state.percent}%`}</Typography>
          <LinearProgress variant="determinate" value={state.percent} />
        </Box>
      )
    default: {
      const label = {
        'waiting-nexus': t`Waiting for Nexus…`,
        queued: t`Queued`,
        installing: t`Installing`,
      }[state.kind]
      return (
        <Box sx={{ width: BAR_PX }}>
          <Typography sx={{ fontSize: 12 }}>{label}</Typography>
          <LinearProgress
            variant={state.kind === 'queued' ? 'determinate' : 'indeterminate'}
            value={0}
          />
        </Box>
      )
    }
  }
}

export { CardProgress }
