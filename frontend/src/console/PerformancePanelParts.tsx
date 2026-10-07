import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableSortLabel,
  Typography,
} from '@mui/material'
import { Copy, Gauge, Play, RefreshCw } from 'lucide-react'
import { type ReactNode, useMemo } from 'react'
import type {
  PerformanceRow,
  SavedReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { cmpText } from '../mods/cmpText.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { space } from '../theme/density.ts'
import { formatTiming } from './formatTiming.ts'
import { CompareTable, ReportSelect } from './PerformanceComparison.tsx'
import type { PanelBusy, SortColumn, SortDirection } from './usePerformancePanel.ts'

function callsLabel(value: number, locale: string) {
  if (value === 0) {
    return '—'
  }
  return Number.isInteger(value) ? value.toLocaleString(locale) : formatTiming(value, locale)
}

const PERCENT = 100

function percentLabel(share: number, locale: string) {
  return `${(share * PERCENT).toLocaleString(locale, { maximumFractionDigits: 1 })}%`
}

interface HeaderProps {
  running: boolean
  busy: PanelBusy
  measuring: boolean
  hasReport: boolean
  onStart: () => void
  onReport: () => void
  onCopy: () => void
  reports: SavedReport[]
  compareId: string
  onCompare: (id: string) => void
  onClearCompare: () => void
  compareSides: boolean
  beforeId: string
  afterId: string
  onBefore: (id: string) => void
  onAfter: (id: string) => void
}

function ComparePickers(props: HeaderProps) {
  const { t } = useLingui()
  const { compareSides, beforeId, afterId, reports, onBefore, onAfter, compareId, onCompare } =
    props
  return compareSides ? (
    <>
      <ReportSelect label={t`Before`} value={beforeId} reports={reports} onChange={onBefore} />
      <ReportSelect label={t`After`} value={afterId} reports={reports} onChange={onAfter} />
    </>
  ) : (
    <ReportSelect
      label={t`Compare with…`}
      value={compareId}
      reports={reports}
      onChange={onCompare}
    />
  )
}

function ReportTable({
  rows,
  sort,
  onSort,
}: {
  rows: PerformanceRow[]
  sort: { column: SortColumn; direction: SortDirection }
  onSort: (column: SortColumn) => void
}) {
  const { t, i18n } = useLingui()
  // A companion's rows carry the 95th percentile and the share of the frame; SMAPI's console report has neither.
  const framed = rows.some((row) => (row.p95Ms ?? 0) > 0 || (row.share ?? 0) > 0)
  const headers: { column: SortColumn; label: string }[] = [
    { column: 'name', label: t`Mod or event` },
    { column: 'averageMs', label: t`Average ms` },
    ...(framed ? [{ column: 'p95Ms' as const, label: t`95th ms` }] : []),
    { column: 'peakMs', label: t`Peak ms` },
    ...(framed ? [{ column: 'share' as const, label: t`% of frame` }] : []),
    { column: 'calls', label: t`Calls` },
  ]
  const orderedRows = useMemo(
    () =>
      [...rows].sort((a, b) => {
        if (sort.column === 'name') {
          const result = cmpText(a.name, b.name)
          return sort.direction === 'asc' ? result : -result
        }
        const result = (a[sort.column] ?? 0) - (b[sort.column] ?? 0)
        return sort.direction === 'asc' ? result : -result
      }),
    [rows, sort],
  )

  return (
    <Box sx={{ maxHeight: 190, overflow: 'auto' }}>
      <Table size="small" stickyHeader={true} aria-label={t`Performance report`}>
        <TableHead>
          <TableRow>
            {headers.map(({ column, label }) => (
              <TableCell
                key={column}
                align={column === 'name' ? 'left' : 'right'}
                sx={{
                  bgcolor: 'var(--mortar-console)',
                  color: 'var(--mortar-ink-sec)',
                  fontSize: 11,
                  whiteSpace: 'nowrap',
                }}
              >
                <TableSortLabel
                  active={sort.column === column}
                  direction={sort.column === column ? sort.direction : 'asc'}
                  onClick={() => onSort(column)}
                  sx={{
                    color: 'inherit',
                    '&.Mui-active': { color: 'var(--mortar-ink)' },
                    '& .MuiTableSortLabel-icon': { color: 'inherit !important' },
                  }}
                >
                  {label}
                </TableSortLabel>
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {orderedRows.map((row) => (
            <TableRow key={row.name} hover={true}>
              <TableCell sx={{ maxWidth: 320, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {row.name}
              </TableCell>
              <TableCell align="right">{formatTiming(row.averageMs, i18n.locale)}</TableCell>
              {framed ? (
                <TableCell align="right">{formatTiming(row.p95Ms ?? 0, i18n.locale)}</TableCell>
              ) : null}
              <TableCell align="right">{formatTiming(row.peakMs, i18n.locale)}</TableCell>
              {framed ? (
                <TableCell align="right">{percentLabel(row.share ?? 0, i18n.locale)}</TableCell>
              ) : null}
              <TableCell align="right">{callsLabel(row.calls, i18n.locale)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  )
}

function PanelHeader(props: HeaderProps) {
  const { t } = useLingui()
  const {
    running,
    busy,
    measuring,
    hasReport,
    onStart,
    onReport,
    onCopy,
    compareId,
    onClearCompare,
    beforeId,
    afterId,
  } = props
  return (
    <Box
      sx={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: space.gap,
        px: space.pad,
        py: space.gap,
      }}
    >
      <Typography sx={{ flex: '1 1 auto', fontSize: 13, fontWeight: 600 }}>
        {t`Performance`}
      </Typography>
      <DisabledReason title={t`Play this profile to measure.`} disabled={!running}>
        <Button
          size="small"
          variant="outlined"
          startIcon={<Play size={14} />}
          disabled={!running || busy !== '' || measuring}
          onClick={onStart}
        >
          {measuring ? t`Measuring` : t`Start measuring`}
        </Button>
      </DisabledReason>
      <DisabledReason
        title={running ? t`Start measuring first.` : t`Play this profile to measure.`}
        disabled={!running || busy !== '' || !measuring}
      >
        <Button
          size="small"
          variant="outlined"
          startIcon={<RefreshCw size={14} />}
          disabled={!running || busy !== '' || !measuring}
          onClick={onReport}
        >
          {t`Show report`}
        </Button>
      </DisabledReason>
      <DisabledReason title={t`Show a report first.`} disabled={!hasReport}>
        <Button
          size="small"
          variant="text"
          color="inherit"
          startIcon={<Copy size={14} />}
          disabled={!hasReport}
          onClick={onCopy}
        >
          {t`Copy report`}
        </Button>
      </DisabledReason>
      <ComparePickers {...props} />
      {(compareId !== '' || (beforeId !== '' && afterId !== '')) && (
        <Button size="small" color="inherit" onClick={onClearCompare}>
          {t`Clear comparison`}
        </Button>
      )}
    </Box>
  )
}

function ReportBody({
  rows,
  reportLines,
  sort,
  onSort,
}: {
  rows: PerformanceRow[]
  reportLines: string[]
  sort: { column: SortColumn; direction: SortDirection }
  onSort: (column: SortColumn) => void
}) {
  const { t } = useLingui()
  return rows.length > 0 ? (
    <ReportTable rows={rows} sort={sort} onSort={onSort} />
  ) : (
    <Typography sx={{ px: space.pad, pb: space.gap, color: 'text.secondary', fontSize: 12 }}>
      {reportLines.length > 0
        ? t`No performance data was returned.`
        : t`Start measuring to view a report.`}
    </Typography>
  )
}

export function PerformanceEmpty({
  running,
  busy,
  onStart,
  reports,
  hint,
  onCompare,
}: {
  running: boolean
  busy: boolean
  onStart: () => void
  reports: SavedReport[]
  hint?: string | undefined
  onCompare: () => void
}) {
  const { t } = useLingui()
  return (
    <EmptyState
      icon={<Gauge size={40} aria-hidden={true} />}
      title={t`See which mods slow the game`}
      action={
        <Box sx={{ display: 'flex', gap: space.gap }}>
          <DisabledReason title={t`Play this profile to measure.`} disabled={!running}>
            <Button
              variant="contained"
              startIcon={<Play size={16} />}
              disabled={!running || busy}
              onClick={onStart}
            >
              {t`Start measuring`}
            </Button>
          </DisabledReason>
          {reports.length >= 2 && (
            <Button variant="outlined" onClick={onCompare}>
              {t`Compare saved reports`}
            </Button>
          )}
        </Box>
      }
    >
      {hint ??
        (running
          ? t`Measure while you play, then report the mods that take the most time per frame.`
          : t`Start the game with this profile, then measure here while you play.`)}
    </EmptyState>
  )
}

export function MeasuredPanel({
  header,
  summary,
  comparing,
  comparisonBefore,
  comparisonNow,
  rows,
  reportLines,
  sort,
  onSort,
}: {
  header: HeaderProps
  summary?: ReactNode
  comparing: boolean
  comparisonBefore: PerformanceRow[]
  comparisonNow: PerformanceRow[]
  rows: PerformanceRow[]
  reportLines: string[]
  sort: { column: SortColumn; direction: SortDirection }
  onSort: (column: SortColumn) => void
}) {
  return (
    <Paper
      variant="outlined"
      sx={{
        mx: space.gutter,
        mb: 1.5,
        flexShrink: 0,
        overflow: 'hidden',
        bgcolor: 'var(--mortar-overlay-24)',
        borderColor: 'var(--mortar-hairline-14)',
      }}
    >
      <PanelHeader {...header} />
      {summary}
      {comparing ? (
        <CompareTable before={comparisonBefore} now={comparisonNow} />
      ) : (
        <ReportBody rows={rows} reportLines={reportLines} sort={sort} onSort={onSort} />
      )}
    </Paper>
  )
}
