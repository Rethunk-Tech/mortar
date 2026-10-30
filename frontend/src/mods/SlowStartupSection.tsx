import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useNowValue } from '@rethunk/react-load-hooks'
import type { CheckTiming } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { CONTENT_PATCHER } from '../brand/productNames.ts'
import { formatDuration, type SlowStartup } from '../console/startupView.ts'
import { useTab } from '../game/tab.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { LockedReason } from './LockedReason.tsx'
import { modById } from './lookup.ts'
import { LinkedText } from './ModLinks.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

function SlowRow({ row }: { row: SlowStartup }) {
  const { t, i18n } = useLingui()
  const mod = useMods((s) => modById(s.mods, row.id))
  const setEnabled = useMods((s) => s.setEnabled)
  const locked = useLocked()
  const time = formatDuration(row.ms, i18n.locale)
  let text = t`${{ name: row.name }} adds ${time} to startup`
  if (row.kind === 'pack') {
    text = t`${{ name: row.name }} adds ${time} to startup (through ${{ framework: row.framework }})`
  }
  return (
    <Box
      role="status"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        pl: space.pad,
        pr: 0.75,
        py: 0.75,
        bgcolor: calloutFill('info'),
        border: '1px solid',
        borderColor: calloutLine('info'),
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ flex: 1, minWidth: 0, fontSize: 'medium.fontSize' }}>
        <LinkedText text={text} links={mod ? [{ name: row.name, key: mod.key, id: mod.id }] : []} />
      </Typography>
      <Button
        size="small"
        variant="text"
        color="inherit"
        onClick={() => useTab.getState().setTab('performance')}
      >
        {t`See startup`}
      </Button>
      {mod?.enabled ? (
        <LockedReason locked={locked}>
          <Button
            size="small"
            variant="outlined"
            color="inherit"
            disabled={locked}
            onClick={() => setEnabled(mod, false).catch(reportUnexpected)}
          >
            {t`Disable`}
          </Button>
        </LockedReason>
      ) : null}
    </Box>
  )
}

/** How long the last Problems check took, by check family, so a slow check is visible rather than guessed at. */
export function CheckTimings({ timings, at }: { timings: CheckTiming[]; at: number }) {
  const { t, i18n } = useLingui()
  // Redraws only when the label changes, not on every tick of the minute clock.
  const when = useNowValue(() => formatWhen(at))
  if (timings.length === 0) {
    return null
  }
  const labels: Record<string, string> = {
    contentPatcher: t`${{ app: CONTENT_PATCHER }} packs`,
    conflicts: lowerFirst(t`Conflicts`),
    requirements: t`requirements`,
    updates: lowerFirst(t`Updates`),
    others: t`other checks`,
  }
  const total = timings.reduce((n, x) => n + x.ms, 0)
  const parts = timings
    .map((x) => `${labels[x.name] ?? x.name} ${formatDuration(x.ms, i18n.locale)}`)
    .join(', ')
  return (
    <Tooltip
      title={t`Checked in ${{ duration: formatDuration(total, i18n.locale) }}: ${parts}`}
      describeChild={true}
    >
      <Typography
        tabIndex={0}
        sx={{ fontSize: 'small.fontSize', color: 'text.secondary', width: 'fit-content' }}
      >
        {t`Checked ${{ when }}`}
      </Typography>
    </Tooltip>
  )
}

export function SlowStartupSection({ rows }: { rows: SlowStartup[] }) {
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {rows.map((row) => (
          <SlowRow key={`${row.kind}/${row.id}`} row={row} />
        ))}
      </Box>
    </Box>
  )
}
