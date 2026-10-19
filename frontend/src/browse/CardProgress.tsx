import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, LinearProgress, Typography } from '@mui/material'
import { Check, RotateCw } from 'lucide-react'
import type { ReactNode } from 'react'
import { Retry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { useQueue } from '../queue/store.ts'
import { BUTTON_SMALL_HEIGHT } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { CardState } from './cardState.ts'

const BAR_PX = 96

// The label and bar fill the Add button's footprint; under a picture the card sizes it to the picture's width.
function Block({
  label,
  title = label,
  children,
}: {
  label: string
  title?: string
  children: ReactNode
}) {
  return (
    <Box
      className="card-progress"
      sx={{
        width: BAR_PX,
        height: BUTTON_SMALL_HEIGHT,
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        minWidth: 0,
      }}
    >
      <Typography noWrap={true} title={title} sx={{ fontSize: 12, lineHeight: 1.2 }}>
        {label}
      </Typography>
      {children}
    </Box>
  )
}

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
          {t`Needs input`}
        </Button>
      )
    case 'downloading':
      return (
        <Block label={`${state.percent}%`} title={t`Downloading ${state.percent}%`}>
          <LinearProgress variant="determinate" value={state.percent} />
        </Block>
      )
    case 'paused':
      return (
        <Block label={t`Paused ${state.percent}%`} title={t`Paused at ${state.percent}%`}>
          <LinearProgress variant="determinate" color="inherit" value={state.percent} />
        </Block>
      )
    default: {
      const label = {
        'waiting-nexus': t`Waiting for Nexus…`,
        queued: t`Queued`,
        installing: t`Installing`,
      }[state.kind]
      return (
        <Block label={label}>
          <LinearProgress
            variant={state.kind === 'queued' ? 'determinate' : 'indeterminate'}
            value={0}
          />
        </Block>
      )
    }
  }
}

export { CardProgress }
