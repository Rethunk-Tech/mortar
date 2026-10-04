import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Collapse,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material'
import { ChevronDown, ChevronRight, Gauge, Timer } from 'lucide-react'
import { useState } from 'react'
import type {
  StartupMod,
  StartupReport,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import {
  foldMods,
  formatDuration,
  modTotal,
  type PhaseId,
  phaseSegments,
  slowestEvent,
} from './startupView.ts'
import { useStartupReports } from './useStartupReports.ts'

const PACKS_SHOWN = 25
const PHASE_COLORS: Record<PhaseId, string> = {
  smapi: 'text.disabled',
  entry: 'primary.main',
  content: 'info.main',
  firstTicks: 'warning.main',
  intro: 'success.main',
}

function useDuration() {
  const { i18n } = useLingui()
  return (ms: number) => formatDuration(ms, i18n.locale)
}

function PhaseBar({ report }: { report: StartupReport }) {
  const { t } = useLingui()
  const duration = useDuration()
  const labels: Record<PhaseId, string> = {
    smapi: t`SMAPI loads mods`,
    entry: t`Mods start`,
    content: t`Game content`,
    firstTicks: t`First updates`,
    intro: t`Title intro`,
  }
  const segments = phaseSegments(report.phases)
  return (
    <Box>
      <Box sx={{ display: 'flex', height: 12, borderRadius: 1, overflow: 'hidden', gap: '2px' }}>
        {segments.map((s) => (
          <Tooltip key={s.id} title={`${labels[s.id]}: ${duration(s.ms)}`}>
            <Box sx={{ flexGrow: s.ms, bgcolor: PHASE_COLORS[s.id] }} />
          </Tooltip>
        ))}
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', columnGap: 2, rowGap: 0.5, mt: 1 }}>
        {segments.map((s) => (
          <Box key={s.id} sx={{ display: 'flex', alignItems: 'center', gap: 0.75, fontSize: 13 }}>
            <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: PHASE_COLORS[s.id] }} />
            <span>{labels[s.id]}</span>
            <Box component="span" sx={{ color: 'text.secondary' }}>
              {duration(s.ms)}
            </Box>
          </Box>
        ))}
      </Box>
    </Box>
  )
}

function ModRow({ mod }: { mod: StartupMod }) {
  const { t } = useLingui()
  const duration = useDuration()
  const [open, setOpen] = useState(false)
  const event = slowestEvent(mod)
  const packs = mod.packs ?? []
  const expandable = packs.length > 0
  return (
    <>
      <TableRow
        hover={expandable}
        onClick={expandable ? () => setOpen((o) => !o) : undefined}
        sx={{ cursor: expandable ? 'pointer' : 'default' }}
      >
        <TableCell sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          {expandable ? (
            <Box
              component="span"
              aria-label={open ? t`Hide content packs` : t`Show content packs`}
              sx={{ display: 'inline-flex' }}
            >
              {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            </Box>
          ) : (
            <Box component="span" sx={{ width: 14 }} />
          )}
          {mod.name}
        </TableCell>
        <TableCell align="right" sx={{ fontWeight: 600 }}>
          {duration(modTotal(mod))}
        </TableCell>
        <TableCell align="right">{mod.entryMs > 0 ? duration(mod.entryMs) : '—'}</TableCell>
        <TableCell align="right">{event ? `${duration(event[1])} · ${event[0]}` : '—'}</TableCell>
        <TableCell align="right">
          {mod.assetMs + mod.loadMs > 0 ? duration(mod.assetMs + mod.loadMs) : '—'}
        </TableCell>
      </TableRow>
      {expandable ? (
        <TableRow>
          <TableCell colSpan={5} sx={{ py: 0, borderBottom: open ? undefined : 'none' }}>
            <Collapse in={open} unmountOnExit={true}>
              <Box
                sx={{
                  py: 1,
                  pl: 3,
                  display: 'grid',
                  gridTemplateColumns: '1fr auto',
                  columnGap: 3,
                  fontSize: 13,
                }}
              >
                {packs.slice(0, PACKS_SHOWN).map((p) => (
                  <Box key={p.id} sx={{ display: 'contents' }}>
                    <span>{p.name}</span>
                    <Box component="span" sx={{ textAlign: 'right', color: 'text.secondary' }}>
                      {duration(p.ms)}
                    </Box>
                  </Box>
                ))}
                {packs.length > PACKS_SHOWN ? (
                  <Box sx={{ gridColumn: '1 / -1', color: 'text.secondary' }}>
                    {plural(packs.length - PACKS_SHOWN, {
                      one: '# more pack',
                      other: '# more packs',
                    })}
                  </Box>
                ) : null}
              </Box>
            </Collapse>
          </TableCell>
        </TableRow>
      ) : null}
    </>
  )
}

function ModTable({ report }: { report: StartupReport }) {
  const { t } = useLingui()
  const duration = useDuration()
  const { shown, folded } = foldMods(report.mods)
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
      <Table size="small" stickyHeader={true} aria-label={t`Startup time by mod`}>
        <TableHead>
          <TableRow>
            <TableCell>{t`Mod`}</TableCell>
            <TableCell align="right">{t`Total`}</TableCell>
            <TableCell align="right">{t`Entry`}</TableCell>
            <TableCell align="right">{t`Slowest event`}</TableCell>
            <TableCell align="right">{t`Assets and packs`}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {shown.map((mod) => (
            <ModRow key={mod.id} mod={mod} />
          ))}
          {folded.count > 0 ? (
            <TableRow>
              <TableCell sx={{ color: 'text.secondary', pl: 4.5 }}>
                {plural(folded.count, { one: '# other mod', other: '# other mods' })}
              </TableCell>
              <TableCell align="right">{duration(folded.ms)}</TableCell>
              <TableCell colSpan={3} />
            </TableRow>
          ) : null}
          <TableRow>
            <TableCell sx={{ color: 'text.secondary', pl: 4.5 }}>{t`Game and SMAPI`}</TableCell>
            <TableCell align="right">{duration(report.otherMs)}</TableCell>
            <TableCell colSpan={3} />
          </TableRow>
        </TableBody>
      </Table>
    </Box>
  )
}

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
          {new Date(r.processStart).toLocaleString(i18n.locale, {
            dateStyle: 'medium',
            timeStyle: 'short',
          })}
        </MenuItem>
      ))}
    </Select>
  )
}

export function StartupPanel({ game }: { game: string }) {
  const { t } = useLingui()
  const duration = useDuration()
  const openId = useProfiles((s) => s.openId)
  const { reports, pending, measureNext } = useStartupReports(game, openId)
  const [selected, setSelected] = useState('')
  const measure = (
    <DisabledReason title={t`The next launch will be measured`} disabled={pending}>
      <Button
        variant="outlined"
        size="small"
        startIcon={<Gauge size={16} />}
        disabled={pending}
        onClick={measureNext}
      >
        {t`Measure next launch`}
      </Button>
    </DisabledReason>
  )
  if (reports === null) {
    return null
  }
  const report = reports.find((r) => r.id === selected) ?? reports[0]
  if (!report) {
    return (
      <EmptyState icon={<Timer size={32} />} title={t`No startup measured yet`} action={measure}>
        {t`Play this profile. The SMAPI Bridge records how long each mod adds before the title screen.`}
      </EmptyState>
    )
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column', gap: 2, p: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
        <Typography variant="h6" sx={{ flex: '1 1 auto' }}>
          {report.phases.titleScreen > 0
            ? t`Title screen after ${duration(report.phases.titleScreen)}`
            : t`The title screen was not reached`}
        </Typography>
        {reports.length > 1 ? (
          <ReportPicker reports={reports} value={report.id} onChange={setSelected} />
        ) : null}
        {measure}
      </Box>
      <PhaseBar report={report} />
      {report.entryTimed ? null : (
        <Typography variant="body2" sx={{ color: 'text.secondary' }}>
          {t`Mods' Entry is timed only on a measured launch.`}
        </Typography>
      )}
      <ModTable report={report} />
    </Box>
  )
}
