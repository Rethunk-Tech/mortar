import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  Collapse,
  IconButton,
  Link,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableSortLabel,
  Tooltip,
  Typography,
} from '@mui/material'
import { ChevronDown, ChevronRight, Gauge, Timer } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import {
  type StartupMod,
  type StartupReport,
  State,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { absoluteWhen } from '../i18n/when.ts'
import { playOpenProfile } from '../launch/playOpen.ts'
import { useLaunch } from '../launch/store.ts'
import { showInProfile } from '../mods/revealMod.ts'
import { useProfiles } from '../profiles/store.ts'
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
  const help: Record<PhaseId, string> = {
    smapi: t`From launch until SMAPI has loaded every mod's code`,
    entry: t`Each mod's Entry method, run one after another`,
    content: t`The game loads its own content`,
    firstTicks: t`The game's first frames; mods doing setup work in update events show up here`,
    intro: t`The title screen's intro animation (Mortar skips it on measured launches)`,
  }
  const segments = phaseSegments(report.phases)
  return (
    <Box>
      <Box sx={{ display: 'flex', height: 12, borderRadius: 1, overflow: 'hidden', gap: '2px' }}>
        {segments.map((s) => (
          <Tooltip key={s.id} title={`${labels[s.id]}: ${duration(s.ms)}. ${help[s.id]}`}>
            <Box sx={{ flexGrow: s.ms, bgcolor: PHASE_COLORS[s.id] }} />
          </Tooltip>
        ))}
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', columnGap: 2, rowGap: 0.5, mt: 1 }}>
        {segments.map((s) => (
          <Tooltip key={s.id} title={help[s.id]}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, fontSize: 13 }}>
              <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: PHASE_COLORS[s.id] }} />
              <span>{labels[s.id]}</span>
              <Box component="span" sx={{ color: 'text.secondary' }}>
                {duration(s.ms)}
              </Box>
            </Box>
          </Tooltip>
        ))}
      </Box>
    </Box>
  )
}

// Mod, Total, Entry, Slowest event, Assets and packs.
const MOD_COLUMNS = 5
const BRIDGE_ID = 'Rethunk.MortarSmapiBridge'

type ModSort = 'name' | 'total' | 'entry' | 'event' | 'assets' | 'sampled'

const sortValue: Record<Exclude<ModSort, 'name'>, (mod: StartupMod) => number> = {
  total: modTotal,
  entry: (mod) => mod.entryMs,
  event: (mod) => slowestEvent(mod)?.[1] ?? 0,
  assets: (mod) => mod.assetMs + mod.loadMs,
  sampled: (mod) => mod.sampleMs,
}

function sortMods(mods: StartupMod[], column: ModSort, direction: 'asc' | 'desc'): StartupMod[] {
  const sign = direction === 'asc' ? 1 : -1
  return [...mods].sort((a, b) =>
    column === 'name'
      ? sign * a.name.localeCompare(b.name)
      : sign * (sortValue[column](a) - sortValue[column](b)),
  )
}

// Clicking the sorted column flips it; another column starts descending, or A to Z for names.
function nextSort(
  active: { column: ModSort; direction: 'asc' | 'desc' },
  column: ModSort,
): { column: ModSort; direction: 'asc' | 'desc' } {
  if (active.column === column) {
    return { column, direction: active.direction === 'desc' ? 'asc' : 'desc' }
  }
  return { column, direction: column === 'name' ? 'asc' : 'desc' }
}

function ModRow({
  mod,
  sampled,
  sort,
  game,
}: {
  mod: StartupMod
  sampled: boolean
  sort: ModSort
  game: string
}) {
  const { t } = useLingui()
  const duration = useDuration()
  const openId = useProfiles((s) => s.openId)
  const [open, setOpen] = useState(false)
  const event = slowestEvent(mod)
  const packs = mod.packs ?? []
  const expandable = packs.length > 0
  const strong = (column: ModSort) => (sort === column ? { fontWeight: 600 } : undefined)
  return (
    <>
      <TableRow
        hover={true}
        onClick={expandable ? () => setOpen((o) => !o) : undefined}
        sx={{ cursor: expandable ? 'pointer' : 'default' }}
      >
        <TableCell sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          {expandable ? (
            <IconButton
              size="small"
              aria-expanded={open}
              aria-label={open ? t`Hide content packs` : t`Show content packs`}
              onClick={(e) => {
                e.stopPropagation()
                setOpen((o) => !o)
              }}
              sx={{ p: 0, borderRadius: '4px', color: 'inherit' }}
            >
              {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            </IconButton>
          ) : (
            <Box component="span" sx={{ width: 14 }} />
          )}
          <Link
            component="button"
            underline="hover"
            color="inherit"
            title={t`Show in Mods`}
            onClick={(e) => {
              e.stopPropagation()
              if (openId !== '') {
                showInProfile(game, openId, 0, mod.name)
              }
            }}
            sx={{
              textAlign: 'left',
              font: 'inherit',
              textDecoration: 'none',
              '&:hover': { textDecoration: 'underline' },
            }}
          >
            {mod.name}
          </Link>
          {mod.id === BRIDGE_ID ? (
            <Tooltip title={t`Mortar's own SMAPI mod; it records these timings`}>
              <Box component="span" sx={{ color: 'text.secondary', fontSize: 12, ml: 0.5 }}>
                {t`Mortar`}
              </Box>
            </Tooltip>
          ) : null}
        </TableCell>
        <TableCell align="right" sx={strong('total')}>
          {duration(modTotal(mod))}
        </TableCell>
        <TableCell align="right" sx={strong('entry')}>
          {mod.entryMs > 0 ? duration(mod.entryMs) : '—'}
        </TableCell>
        <TableCell align="right" sx={strong('event')}>
          {event ? `${duration(event[1])} · ${event[0]}` : '—'}
        </TableCell>
        <TableCell align="right" sx={strong('assets')}>
          {mod.assetMs + mod.loadMs > 0 ? duration(mod.assetMs + mod.loadMs) : '—'}
        </TableCell>
        {sampled ? (
          <TableCell align="right" sx={strong('sampled')}>
            {mod.sampleMs ? duration(mod.sampleMs) : '—'}
          </TableCell>
        ) : null}
      </TableRow>
      {expandable ? (
        <TableRow>
          <TableCell
            colSpan={sampled ? MOD_COLUMNS + 1 : MOD_COLUMNS}
            sx={{ py: 0, borderBottom: open ? undefined : 'none' }}
          >
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

function ModTable({ report, game }: { report: StartupReport; game: string }) {
  const { t } = useLingui()
  const duration = useDuration()
  const { shown, folded } = foldMods(report.mods)
  // Sampled times exist only after a measured launch; they include time inside each mod's patches on game code.
  const sampled = report.sampledOtherMs > 0
  const [sort, setSort] = useState<{ column: ModSort; direction: 'asc' | 'desc' } | null>(null)
  const active = sort ?? { column: 'total' as ModSort, direction: 'desc' as const }
  const rows = sortMods(shown, active.column, active.direction)
  const shownIds = new Set(shown.map((mod) => mod.id))
  const foldedSampled = (report.mods ?? []).reduce(
    (sum, mod) => (shownIds.has(mod.id) ? sum : sum + mod.sampleMs),
    0,
  )
  const columns: { id: ModSort; label: string; help: string; align?: 'right' }[] = [
    { id: 'name', label: t`Mod`, help: t`Click a mod to show it in Mods` },
    {
      id: 'total',
      label: t`Total`,
      help: t`Entry, event handlers and asset work the SMAPI Bridge timed before the title screen`,
      align: 'right',
    },
    {
      id: 'entry',
      label: t`Entry`,
      help: t`The mod's Entry method; timed only on a measured launch`,
      align: 'right',
    },
    {
      id: 'event',
      label: t`Slowest event`,
      help: t`The SMAPI event handler that took longest, and its time`,
      align: 'right',
    },
    {
      id: 'assets',
      label: t`Assets and packs`,
      help: t`Time editing and loading game assets, including content packs it loads`,
      align: 'right',
    },
    ...(sampled
      ? [
          {
            id: 'sampled' as const,
            label: t`Sampled`,
            help: t`Sampled on a measured launch: includes time in the mod's patches on game code`,
            align: 'right' as const,
          },
        ]
      : []),
  ]
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto' }}>
      <Table size="small" stickyHeader={true} aria-label={t`Startup time by mod`}>
        <TableHead>
          <TableRow>
            {columns.map((column) => (
              <TableCell
                key={column.id}
                align={column.align}
                sortDirection={active.column === column.id ? active.direction : false}
              >
                <Tooltip title={column.help}>
                  <TableSortLabel
                    active={active.column === column.id}
                    direction={active.column === column.id ? active.direction : 'desc'}
                    onClick={() => setSort(nextSort(active, column.id))}
                    sx={{
                      color: 'inherit',
                      '&.Mui-active': { color: 'var(--mortar-ink)' },
                      '& .MuiTableSortLabel-icon': { color: 'inherit !important' },
                    }}
                  >
                    {column.label}
                  </TableSortLabel>
                </Tooltip>
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((mod) => (
            <ModRow key={mod.id} mod={mod} sampled={sampled} sort={active.column} game={game} />
          ))}
          {folded.count > 0 ? (
            <TableRow>
              <TableCell sx={{ color: 'text.secondary', pl: 4.5 }}>
                {plural(folded.count, { one: '# other mod', other: '# other mods' })}
              </TableCell>
              <TableCell align="right">{duration(folded.ms)}</TableCell>
              <TableCell colSpan={3} />
              {sampled ? <TableCell align="right">{duration(foldedSampled)}</TableCell> : null}
            </TableRow>
          ) : null}
          <TableRow>
            <TableCell sx={{ color: 'text.secondary', pl: 4.5 }}>{t`Game and SMAPI`}</TableCell>
            <TableCell align="right">{duration(report.otherMs)}</TableCell>
            <TableCell colSpan={3} />
            {sampled ? (
              <TableCell align="right">{duration(report.sampledOtherMs)}</TableCell>
            ) : null}
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
          {absoluteWhen(r.processStart, i18n.locale)}
        </MenuItem>
      ))}
    </Select>
  )
}

function Stat({ label, value, note }: { label: string; value: string; note?: string }) {
  return (
    <Box
      sx={{
        px: 1.75,
        py: 1,
        borderRadius: '8px',
        bgcolor: 'var(--mortar-overlay-30)',
        border: '1px solid var(--mortar-hairline-12)',
        minWidth: 0,
      }}
    >
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{label}</Typography>
      <Typography
        noWrap={true}
        sx={{ fontSize: 20, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}
      >
        {value}
      </Typography>
      {note ? (
        <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary', maxWidth: 220 }}>
          {note}
        </Typography>
      ) : null}
    </Box>
  )
}

// The launch at a glance: time to the title screen, the change since the launch before it, and the slowest mod.
function Summary({
  report,
  previous,
}: {
  report: StartupReport
  previous: StartupReport | undefined
}) {
  const { t } = useLingui()
  const duration = useDuration()
  const title = report.phases.titleScreen
  const [slowest] = [...(report.mods ?? [])].sort((a, b) => modTotal(b) - modTotal(a))
  const before = previous?.phases.titleScreen ?? 0
  const change = title > 0 && before > 0 ? title - before : null
  return (
    <>
      <Stat label={t`Title screen`} value={title > 0 ? duration(title) : t`Not reached`} />
      {change === null ? null : (
        <Stat
          label={t`Since the launch before`}
          value={duration(Math.abs(change))}
          note={change > 0 ? t`slower` : t`faster`}
        />
      )}
      {slowest ? (
        <Stat label={t`Slowest mod`} value={duration(modTotal(slowest))} note={slowest.name} />
      ) : null}
      <Stat label={t`Mods measured`} value={String((report.mods ?? []).length)} />
    </>
  )
}

function MeasureBanner({ onCancel }: { onCancel: () => void }) {
  const { t } = useLingui()
  const running = useLaunch(
    (s) => s.starting || s.status?.state === State.Launching || s.status?.state === State.Running,
  )
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
      {t`Press Play. Mortar samples the game until the title screen and skips the intro animation; the results, with a Sampled column, appear here when the title screen is reached.`}
    </Alert>
  )
}

export function StartupPanel({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const { reports, pending, measureNext, cancelMeasure } = useStartupReports(game, openId)
  const [selected, setSelected] = useState('')
  const measure = pending ? null : (
    <Button variant="outlined" size="small" startIcon={<Gauge size={16} />} onClick={measureNext}>
      {t`Measure next launch`}
    </Button>
  )
  const banner: ReactNode = pending ? <MeasureBanner onCancel={cancelMeasure} /> : null
  if (reports === null) {
    return null
  }
  const report = reports.find((r) => r.id === selected) ?? reports[0]
  if (!report) {
    return (
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
          p: banner ? 2 : 0,
        }}
      >
        {banner}
        <EmptyState icon={<Timer size={32} />} title={t`No startup measured yet`} action={measure}>
          {t`Play this profile. The SMAPI Bridge records how long each mod adds before the title screen.`}
        </EmptyState>
      </Box>
    )
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column', gap: 2, p: 2 }}>
      {banner}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', flex: '1 1 auto' }}>
          <Summary report={report} previous={reports[reports.indexOf(report) + 1]} />
        </Box>
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
      <ModTable report={report} game={game} />
    </Box>
  )
}
