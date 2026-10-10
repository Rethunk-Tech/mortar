import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Collapse, MenuItem, Select, Typography } from '@mui/material'
import { ChevronDown, ChevronRight, Gauge } from 'lucide-react'
import { useState } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { PageActions } from '../game/PageActions.tsx'
import { absoluteWhen } from '../i18n/when.ts'
import { useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { PerformancePanel } from './PerformancePanel.tsx'
import { StartupPanel } from './StartupPanel.tsx'
import { usePerfQuery, useSmapiStartup } from './startupHooks.ts'
import { pickReport } from './startupView.ts'
import { useSavedReports } from './usePerformancePanel.ts'
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

// The in-game reports are saved per profile, so a profile with none starts with the section closed.
function InGame({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const [saved] = useSavedReports(game, openId)
  const [choice, setChoice] = useState<boolean | null>(null)
  const open = choice ?? saved.length > 0
  return (
    <Box component="section">
      {/* The button sits inside the heading, so the section is still found by heading navigation. */}
      <Typography component="h2" sx={{ m: 0, fontSize: 18, fontWeight: 600 }}>
        <ButtonBase
          aria-expanded={open}
          onClick={() => setChoice(!open)}
          sx={{ gap: 0.75, borderRadius: '4px', justifyContent: 'flex-start', font: 'inherit' }}
        >
          {open ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          {t`In game`}
        </ButtonBase>
      </Typography>
      <Collapse in={open} unmountOnExit={true}>
        <PerformancePanel game={game} />
      </Collapse>
    </Box>
  )
}

// In game reads SMAPI's performance counters through its console, or a companion that measures in game.
export function PerformanceTab({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const smapi = useSmapiStartup()
  const perf = usePerfQuery()
  const { reports, pending, measureNext, cancelMeasure } = useStartupReports(game, openId)
  const [selected, setSelected] = useState('')
  const report = reports ? pickReport(reports, selected) : undefined
  const actions =
    reports === null ? null : (
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
      <StartupPanel
        reports={reports}
        pending={pending}
        cancelMeasure={cancelMeasure}
        selected={selected}
      >
        {smapi || perf ? <InGame game={game} /> : null}
      </StartupPanel>
    </Box>
  )
}
