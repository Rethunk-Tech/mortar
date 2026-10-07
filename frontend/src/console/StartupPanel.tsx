import { useLingui } from '@lingui/react/macro'
import { Alert, AlertTitle, Box, Button, MenuItem, Select, Typography } from '@mui/material'
import { Gauge, Timer } from 'lucide-react'
import { type ReactNode, useRef, useState } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { absoluteWhen } from '../i18n/when.ts'
import { playOpenProfile } from '../launch/playOpen.ts'
import { useGameBusy } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { Findings, Phases, Section } from './StartupSections.tsx'
import { ModTable } from './StartupTable.tsx'
import { useSmapiStartup } from './startupHooks.ts'
import { type Finding, rowAnchor, startupFindings } from './startupView.ts'
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

function MeasureBanner({ onCancel }: { onCancel: () => void }) {
  const { t } = useLingui()
  const running = useGameBusy()
  const smapi = useSmapiStartup()
  return (
    <Alert
      severity="info"
      icon={<Gauge size={18} aria-hidden={true} />}
      action={
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
          <Button size="small" variant="contained" disabled={running} onClick={playOpenProfile}>
            {t`Play now`}
          </Button>
          <Button size="small" color="inherit" onClick={onCancel}>
            {t`Cancel`}
          </Button>
        </Box>
      }
    >
      <AlertTitle>{t`The next launch will be measured`}</AlertTitle>
      {smapi
        ? t`Press Play. Mortar samples the game until the title screen, skipping the intro animation, and shows the results here, with a Sampled column.`
        : t`Press Play. The BepInEx Bridge times each plugin's load and the start up to the main menu, and shows the results here. Unity's Mono runtime cannot be sampled, so there is no Sampled column.`}
    </Alert>
  )
}

export function StartupPanel({ game, children }: { game: string; children: ReactNode }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const { reports, pending, measureNext, cancelMeasure } = useStartupReports(game, openId)
  const smapi = useSmapiStartup()
  const [selected, setSelected] = useState('')
  const modsSection = useRef<HTMLElement>(null)
  const [comparing, setComparing] = useState(false)
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set())
  const measure = pending ? null : (
    <Button variant="outlined" size="small" startIcon={<Gauge size={16} />} onClick={measureNext}>
      {t`Measure next launch`}
    </Button>
  )
  const banner: ReactNode = pending ? <MeasureBanner onCancel={cancelMeasure} /> : null
  if (reports === null) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, p: 2 }}>
        <SkeletonRows label={t`Reading startup reports…`} count={1} height={72} />
        <SkeletonRows label={t`Reading startup reports…`} count={6} height={36} />
      </Box>
    )
  }
  const report = reports.find((r) => r.id === selected) ?? reports[0]
  const previous = report ? reports[reports.indexOf(report) + 1] : undefined
  const toggle = (id: string) =>
    setExpanded((cur) => {
      const next = new Set(cur)
      if (!next.delete(id)) {
        next.add(id)
      }
      return next
    })
  const scrollTo = (anchor: string) =>
    requestAnimationFrame(() =>
      document.getElementById(anchor)?.scrollIntoView({ block: 'center', behavior: 'smooth' }),
    )
  const act = (finding: Finding) => {
    if (finding.kind === 'heavy') {
      setComparing(false)
      setExpanded((cur) => new Set(cur).add(finding.mod.id))
      scrollTo(rowAnchor(finding.mod.id))
      return
    }
    setComparing(finding.kind === 'slower')
    modsSection.current?.scrollIntoView({ block: 'start', behavior: 'smooth' })
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, p: 2 }}>
      {banner}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
        <Typography component="h2" sx={{ fontSize: 20, fontWeight: 700, flex: '1 1 auto' }}>
          {t`Startup`}
        </Typography>
        {report && reports.length > 1 ? (
          <ReportPicker reports={reports} value={report.id} onChange={setSelected} />
        ) : null}
        {report ? measure : null}
      </Box>
      {report ? (
        <>
          <Findings
            findings={startupFindings(report, previous).filter((f) => smapi || f.kind !== 'smapi')}
            onAct={act}
          />
          <Phases report={report} />
          <Section title={t`Slowest mods`} anchor={modsSection}>
            {report.entryTimed ? null : (
              <Typography variant="body2" sx={{ color: 'text.secondary' }}>
                {t`Mods' Entry is timed only on a measured launch.`}
              </Typography>
            )}
            <ModTable
              report={report}
              previous={previous}
              comparing={comparing}
              expanded={expanded}
              onToggle={toggle}
              onSorted={() => setComparing(false)}
            />
          </Section>
        </>
      ) : (
        <EmptyState
          compact={true}
          icon={<Timer size={32} />}
          title={t`No startup measured yet`}
          action={measure}
        >
          {smapi
            ? t`Play this profile. The SMAPI Bridge records how long each mod adds before the title screen.`
            : t`Measure a launch. The BepInEx Bridge records how long each plugin takes to load before the main menu.`}
        </EmptyState>
      )}
      {children}
    </Box>
  )
}
