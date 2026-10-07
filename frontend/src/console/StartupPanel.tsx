import { useLingui } from '@lingui/react/macro'
import { Alert, AlertTitle, Box, Button, Typography } from '@mui/material'
import { Gauge, Timer } from 'lucide-react'
import { type ReactNode, useRef, useState } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { playOpenProfile } from '../launch/playOpen.ts'
import { useGameBusy } from '../launch/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { space } from '../theme/density.ts'
import { Findings, Phases, Section } from './StartupSections.tsx'
import { ModTable } from './StartupTable.tsx'
import { useSmapiStartup } from './startupHooks.ts'
import { type Finding, pickReport, rowAnchor, startupFindings } from './startupView.ts'

function MeasureBanner({ onCancel }: { onCancel: () => void }) {
  const { t } = useLingui()
  const running = useGameBusy()
  const smapi = useSmapiStartup()
  return (
    <Alert
      severity="info"
      icon={<Gauge size={18} aria-hidden={true} />}
      action={
        <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center' }}>
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

export function StartupPanel({
  reports,
  pending,
  cancelMeasure,
  selected,
  children,
}: {
  reports: StartupReport[] | null
  pending: boolean
  cancelMeasure: () => void
  selected: string
  children: ReactNode
}) {
  const { t } = useLingui()
  const smapi = useSmapiStartup()
  const modsSection = useRef<HTMLElement>(null)
  const [comparing, setComparing] = useState(false)
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set())
  const banner: ReactNode = pending ? <MeasureBanner onCancel={cancelMeasure} /> : null
  if (reports === null) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad, p: space.pad }}>
        <SkeletonRows label={t`Reading startup reports…`} count={1} height={72} />
        <SkeletonRows label={t`Reading startup reports…`} count={6} height={36} />
      </Box>
    )
  }
  const report = pickReport(reports, selected)
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
    <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: space.pad, p: space.pad }}>
      {banner}
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
        <Box sx={{ flex: 1, display: 'grid', gridTemplateRows: '1fr auto 1fr', gap: space.pad }}>
          <Box sx={{ gridRow: 2, display: 'flex' }}>
            <EmptyState icon={<Timer size={40} />} title={t`No startup measured yet`}>
              {smapi
                ? t`Play this profile. The SMAPI Bridge records how long each mod adds before the title screen.`
                : t`Measure a launch. The BepInEx Bridge records how long each plugin takes to load before the main menu.`}
            </EmptyState>
          </Box>
          <Box sx={{ gridRow: 3, alignSelf: 'end' }}>{children}</Box>
        </Box>
      )}
      {report ? children : null}
    </Box>
  )
}
