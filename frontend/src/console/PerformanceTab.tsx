import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select } from '@mui/material'
import { absoluteWhen } from '@rethunk/react-load-hooks'
import { Gauge } from 'lucide-react'
import { useState } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { PerformanceCards, type PerformanceView } from './PerformanceCards.tsx'
import { PerformancePanel } from './PerformancePanel.tsx'
import { StartupPanel } from './StartupPanel.tsx'
import { usePerfQuery, useSmapiStartup } from './startupHooks.ts'
import { pickReport, startupFindings } from './startupView.ts'
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
  const [choice, setChoice] = useState<PerformanceView | null>(null)
  const inGame = smapi || perf
  const view: PerformanceView = inGame ? (choice ?? (running ? 'inGame' : 'startup')) : 'startup'
  const report = reports ? pickReport(reports, selected) : undefined
  const previous = report && reports ? reports[reports.indexOf(report) + 1] : undefined
  const calm =
    report !== undefined &&
    startupFindings(report, previous).filter((f) => smapi || f.kind !== 'smapi').length === 0
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
      {reports === null ? null : (
        <Box sx={{ px: space.pad, pt: space.pad }}>
          <PerformanceCards
            view={view}
            onChoose={setChoice}
            report={report}
            calm={calm}
            inGame={inGame}
            saved={saved.length}
            running={running}
          />
        </Box>
      )}
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
