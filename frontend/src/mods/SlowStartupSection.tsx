import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { alpha, useTheme } from '@mui/material/styles'
import { useEffect, useState } from 'react'
import { StartupReports } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { CheckTiming } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { formatDuration, type SlowStartup, slowStartups } from '../console/startupView.ts'
import { useTab } from '../game/tab.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

const INFO_FILL = 0.12
const INFO_LINE = 0.45

function useSlowStartups(): SlowStartup[] {
  const game = useProfiles((s) => s.game?.id ?? '')
  const profileId = useProfiles((s) => s.openId)
  const [rows, setRows] = useState<SlowStartup[]>([])
  useEffect(() => {
    if (game === '' || profileId === '') {
      return
    }
    StartupReports(game, profileId)
      .then((reports) => setRows(reports?.[0] ? slowStartups(reports[0]) : []))
      .catch(reportUnexpected)
  }, [game, profileId])
  return rows
}

function SlowRow({ row }: { row: SlowStartup }) {
  const { t, i18n } = useLingui()
  const info = useTheme().palette.info.main
  const mod = useMods((s) =>
    s.mods.find((m) => m.uniqueId.toLowerCase() === row.uniqueId.toLowerCase()),
  )
  const setEnabled = useMods((s) => s.setEnabled)
  const time = formatDuration(row.ms, i18n.locale)
  let text = t`${row.name} adds ${time} to startup`
  if (row.kind === 'packs') {
    text = t`${row.name} spends ${time} at startup on ${plural(row.count, { one: '# content pack', other: '# content packs' })}`
  } else if (row.kind === 'pack') {
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
        bgcolor: alpha(info, INFO_FILL),
        border: `1px solid ${alpha(info, INFO_LINE)}`,
        borderRadius: '6px',
      }}
    >
      <Typography sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>{text}</Typography>
      <Button
        size="small"
        variant="text"
        color="inherit"
        onClick={() => useTab.getState().setTab('performance')}
      >
        {t`See startup`}
      </Button>
      {row.kind !== 'packs' && mod?.enabled ? (
        <Button
          size="small"
          variant="outlined"
          color="inherit"
          onClick={() => setEnabled(mod, false).catch(reportUnexpected)}
        >
          {t`Switch off`}
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
    conflicts: t`conflicts`,
    requirements: t`requirements`,
    updates: t`updates`,
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
  const { t } = useLingui()
  const rows = useSlowStartups()
  if (rows.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography sx={{ mb: 1, fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
        {t`Slow startup`}
      </Typography>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {rows.map((row) => (
          <SlowRow key={`${row.kind}/${row.uniqueId}`} row={row} />
        ))}
      </Box>
    </Box>
  )
}
