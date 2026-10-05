import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { CheckTiming } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { formatDuration, type SlowStartup } from '../console/startupView.ts'
import { useTab } from '../game/tab.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { sameId } from './lookup.ts'
import { LinkedText } from './ModLinks.tsx'
import { useSlowStartups } from './slowStartups.ts'
import { useMods } from './store.ts'

function SlowRow({ row }: { row: SlowStartup }) {
  const { t, i18n } = useLingui()
  const mod = useMods((s) => s.mods.find((m) => sameId(m.id, row.id)))
  const setEnabled = useMods((s) => s.setEnabled)
  const time = formatDuration(row.ms, i18n.locale)
  let text = t`${row.name} adds ${time} to startup`
  if (row.kind === 'pack') {
    text = t`${row.name} adds ${time} to startup (through ${row.framework})`
  }
  return (
    <Box
      role="status"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        pl: 1.5,
        pr: 0.75,
        py: 0.75,
        bgcolor: calloutFill('info'),
        border: '1px solid',
        borderColor: calloutLine('info'),
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
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
        <Button
          size="small"
          variant="outlined"
          color="inherit"
          onClick={() => setEnabled(mod, false).catch(reportUnexpected)}
        >
          {t`Disable`}
        </Button>
      ) : null}
    </Box>
  )
}

/** How long the last Problems check took, by check family, so a slow check is visible rather than guessed at. */
export function CheckTimings({ timings }: { timings: CheckTiming[] }) {
  const { t, i18n } = useLingui()
  if (timings.length === 0) {
    return null
  }
  const labels: Record<string, string> = {
    contentPatcher: t`Content Patcher packs`,
    conflicts: t`Conflicts`.toLowerCase(),
    requirements: t`requirements`,
    updates: t`Updates`.toLowerCase(),
    others: t`other checks`,
  }
  const total = timings.reduce((n, x) => n + x.ms, 0)
  const parts = timings
    .map((x) => `${labels[x.name] ?? x.name} ${formatDuration(x.ms, i18n.locale)}`)
    .join(', ')
  return (
    <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
      {t`Checked in ${formatDuration(total, i18n.locale)}: ${parts}`}
    </Typography>
  )
}

export function SlowStartupSection() {
  const rows = useSlowStartups()
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {rows.map((row) => (
          <SlowRow key={`${row.kind}/${row.id}`} row={row} />
        ))}
      </Box>
    </Box>
  )
}
