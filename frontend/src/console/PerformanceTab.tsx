import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select } from '@mui/material'
import { Gauge } from 'lucide-react'
import { useState } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { PageActions } from '../game/PageActions.tsx'
import { absoluteWhen } from '../i18n/when.ts'
import { useProfiles } from '../profiles/store.ts'
import { SectionStrip } from '../shell/SectionStrip.tsx'
import { space } from '../theme/density.ts'
import { PerformancePanel } from './PerformancePanel.tsx'
import { StartupPanel } from './StartupPanel.tsx'
import { usePerfQuery, useSmapiStartup } from './startupHooks.ts'
import { pickReport } from './startupView.ts'
import { usePerformanceRunning, useSavedReports } from './usePerformancePanel.ts'
import { useStartupReports } from './useStartupReports.ts'

function ReportPicker({
  reports,
  value,
  onChange,
}: {
  reports: StartupReport[]
  value: string
  onChange: (id: string) => void
}) {
  const { t, i18n } = useLingui()
  return (
    <Select
      size="small"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      inputProps={{ 'aria-label': t`Launch` }}
    >
      {reports.map((r) => (
        <MenuItem key={r.id} value={r.id}>
          {absoluteWhen(r.processStart, i18n.locale)}
        </MenuItem>
      ))}
    </Select>
  )
}

type View = 'startup' | 'inGame'

// In game reads SMAPI's performance counters through its console, or a companion that measures in game. It opens while
// the game runs, since that is the only time it can measure, and Startup otherwise; a choice holds until the next visit.
export function PerformanceTab({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const smapi = useSmapiStartup()
  const perf = usePerfQuery()
  const running = usePerformanceRunning(game)
  const [saved] = useSavedReports(game, openId)
  const { reports, pending, measureNext, cancelMeasure } = useStartupReports(game, openId)
  const [selected, setSelected] = useState('')
  const [choice, setChoice] = useState<View | null>(null)
  const inGame = smapi || perf
  const view: View = inGame ? (choice ?? (running ? 'inGame' : 'startup')) : 'startup'
  const report = reports ? pickReport(reports, selected) : undefined
  const actions =
    view !== 'startup' || reports === null ? null : (
      <>
        {report && reports.length > 1 ? (
          <ReportPicker reports={reports} value={report.id} onChange={setSelected} />
        ) : null}
        {pending ? null : (
          <Button
            variant="outlined"
            color="inherit"
            startIcon={<Gauge size={16} />}
            onClick={measureNext}
            sx={{ height: space.control, borderColor: 'var(--mortar-hairline-20)' }}
          >
            {t`Measure next launch`}
          </Button>
        )}
      </>
    )
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      <PageActions>{actions}</PageActions>
      {inGame ? (
        <SectionStrip
          label={t`Performance views`}
          tabs={[
            { id: 'startup', label: t`Startup`, count: reports?.length ?? 0, errors: false },
            { id: 'inGame', label: t`In game`, count: saved.length, errors: false },
          ]}
          current={view}
          onChoose={(id) => setChoice(id === 'inGame' ? 'inGame' : 'startup')}
          actions={null}
        />
      ) : null}
      {view === 'inGame' ? (
        <PerformancePanel game={game} />
      ) : (
        <StartupPanel
          reports={reports}
          pending={pending}
          cancelMeasure={cancelMeasure}
          selected={selected}
        />
      )}
    </Box>
  )
}
