import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Stack, Typography } from '@mui/material'
import type { Run } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import type { HealthPoint } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  buildHealthSparklinePath,
  HEALTH_SPARKLINE_HEIGHT,
  HEALTH_SPARKLINE_WIDTH,
  sparklineSeries,
} from './healthSparkline.ts'

const RUN_OUTCOMES_SHOWN = 5
const RUN_DOT = 8
const RUN_DOT_GAP = 0.5

type RunKind = 'crashed' | 'errors' | 'clean'

function runKind(run: Run): RunKind {
  if (run.outcome === 'crashed') {
    return 'crashed'
  }
  if ((run.errors ?? 0) > 0) {
    return 'errors'
  }
  return 'clean'
}

function runColor(kind: RunKind): string {
  switch (kind) {
    case 'crashed':
      return '#f44336'
    case 'errors':
      return '#ff9800'
    default:
      return '#66bb6a'
  }
}

function runLabel(i18n: I18n, kind: RunKind): string {
  switch (kind) {
    case 'crashed':
      return i18n._(msg`Crashed`)
    case 'errors':
      return i18n._(msg`Errors`)
    default:
      return i18n._(msg`Clean`)
  }
}

export function HealthTooltipContent({
  summary,
  history,
  runs,
}: {
  summary: string
  history: HealthPoint[]
  runs: Run[]
}) {
  const { i18n } = useLingui()
  const values = sparklineSeries(
    history.map((p) => ({ problems: p.problems ?? 0, updates: p.updates ?? 0 })),
  )
  const path = buildHealthSparklinePath(values)
  const recentRuns = runs.slice(0, RUN_OUTCOMES_SHOWN)

  return (
    <Stack spacing={1} sx={{ maxWidth: 200 }}>
      <Typography variant="caption" sx={{ whiteSpace: 'pre-line' }}>
        {summary}
      </Typography>
      {path === '' ? null : (
        <Box
          component="svg"
          width={HEALTH_SPARKLINE_WIDTH}
          height={HEALTH_SPARKLINE_HEIGHT}
          aria-hidden={true}
        >
          <path d={path} fill="none" stroke="currentColor" strokeWidth={1.5} />
        </Box>
      )}
      {recentRuns.length > 0 ? (
        <Stack direction="row" spacing={RUN_DOT_GAP} sx={{ alignItems: 'center' }}>
          {recentRuns.map((run) => {
            const kind = runKind(run)
            return (
              <Box
                key={run.id}
                role="img"
                aria-label={runLabel(i18n, kind)}
                sx={{
                  width: RUN_DOT,
                  height: RUN_DOT,
                  borderRadius: '50%',
                  bgcolor: runColor(kind),
                }}
              />
            )
          })}
        </Stack>
      ) : null}
    </Stack>
  )
}
