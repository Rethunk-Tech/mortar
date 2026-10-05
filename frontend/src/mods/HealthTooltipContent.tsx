import type { I18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Stack, Typography } from '@mui/material'
import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import type { HealthPoint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  buildHealthSparklinePath,
  HEALTH_SPARKLINE_HEIGHT,
  HEALTH_SPARKLINE_WIDTH,
  sparklineSeries,
} from './healthSparkline.ts'

const RUN_OUTCOMES_SHOWN = 5

function runSummary(i18n: I18n, runs: Run[]): string {
  const crashed = runs.filter((r) => r.outcome === 'crashed').length
  const errors = runs.filter((r) => r.outcome !== 'crashed' && (r.errors ?? 0) > 0).length
  const clean = runs.length - crashed - errors
  const parts = [
    crashed > 0
      ? i18n._(msg`${plural(crashed, { one: '# crashed run', other: '# crashed runs' })}`)
      : '',
    errors > 0 ? i18n._(msg`${errors} with errors`) : '',
    clean > 0 ? i18n._(msg`${clean} clean`) : '',
  ].filter((p) => p !== '')
  const list = parts.join(', ')
  return plural(runs.length, {
    one: `Last run: ${list}`,
    other: `Last # runs: ${list}`,
  })
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
      {/* A flat line says nothing, so the trend shows only when the count moved. */}
      {path === '' || new Set(values).size < 2 ? null : (
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
        <Typography variant="caption" sx={{ color: 'text.secondary' }}>
          {runSummary(i18n, recentRuns)}
        </Typography>
      ) : null}
    </Stack>
  )
}
