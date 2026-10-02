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
import { Clipboard } from '@wailsio/runtime'
import { Copy, Play, RefreshCw } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type { PerformanceRow } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { PerformanceReport } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { canSendTo, useConsole } from './store.ts'

const ENABLE_COMMAND = 'performance enable'
const SUMMARY_COMMAND = 'performance summary'
const REPORT_TITLE = /summary|performance counter/i

type SortColumn = 'name' | 'averageMs' | 'peakMs' | 'calls'
type SortDirection = 'asc' | 'desc'

function numberLabel(value: number) {
  return value.toFixed(2)
}

function callsLabel(value: number) {
  if (value === 0) {
    return '—'
  }
  return Number.isInteger(value) ? value.toLocaleString() : value.toFixed(2)
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
  const { t } = useLingui()
  const headers: { column: SortColumn; label: string }[] = [
    { column: 'name', label: t`Mod or event` },
    { column: 'averageMs', label: t`Average ms` },
    { column: 'peakMs', label: t`Peak ms` },
    { column: 'calls', label: t`Calls` },
  ]
  const orderedRows = useMemo(
    () =>
      [...rows].sort((a, b) => {
        if (sort.column === 'name') {
          const result = a.name.localeCompare(b.name)
          return sort.direction === 'asc' ? result : -result
        }
        const result = a[sort.column] - b[sort.column]
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
                  bgcolor: 'rgba(25,25,30,0.96)',
                  color: 'rgba(230,230,235,0.8)',
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
                    '&.Mui-active': { color: '#ffffff' },
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
              <TableCell align="right">{numberLabel(row.averageMs)}</TableCell>
              <TableCell align="right">{numberLabel(row.peakMs)}</TableCell>
              <TableCell align="right">{callsLabel(row.calls)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  )
}

function PanelHeader({
  running,
  busy,
  measuring,
  hasReport,
  onStart,
  onReport,
  onCopy,
}: {
  running: boolean
  busy: 'start' | 'report' | ''
  measuring: boolean
  hasReport: boolean
  onStart: () => void
  onReport: () => void
  onCopy: () => void
}) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: 1,
        px: 1.5,
        py: 1,
      }}
    >
      <Typography sx={{ flex: '1 1 auto', fontSize: 13, fontWeight: 600 }}>
        {t`Performance`}
      </Typography>
      <Button
        size="small"
        variant="outlined"
        startIcon={<Play size={14} />}
        disabled={!running || busy !== '' || measuring}
        onClick={onStart}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {measuring ? t`Measuring` : t`Start measuring`}
      </Button>
      <Button
        size="small"
        variant="outlined"
        startIcon={<RefreshCw size={14} />}
        disabled={!running || busy !== '' || !measuring}
        onClick={onReport}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {t`Show report`}
      </Button>
      <Button
        size="small"
        variant="text"
        color="inherit"
        startIcon={<Copy size={14} />}
        disabled={!hasReport}
        onClick={onCopy}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {t`Copy report`}
      </Button>
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
    <Typography sx={{ px: 1.5, pb: 1, color: 'text.secondary', fontSize: 12 }}>
      {reportLines.length > 0
        ? t`No performance data was returned.`
        : t`Start measuring to view a report.`}
    </Typography>
  )
}

export function PerformancePanel({ game }: { game: string }) {
  const { t } = useLingui()
  const entries = useConsole((s) => s.entries)
  const send = useConsole((s) => s.send)
  const viewingRun = useConsole((s) => s.viewingRun)
  const status = useLaunch((s) => s.status)
  const openId = useProfiles((s) => s.openId)
  const running = viewingRun === '' && canSendTo(status, game, openId)
  const [measuring, setMeasuring] = useState(false)
  const [busy, setBusy] = useState<'start' | 'report' | ''>('')
  const [reportAfter, setReportAfter] = useState<number | null>(null)
  const [rows, setRows] = useState<PerformanceRow[]>([])
  const [sort, setSort] = useState<{ column: SortColumn; direction: SortDirection }>({
    column: 'peakMs',
    direction: 'desc',
  })

  useEffect(() => {
    if (!running) {
      setMeasuring(false)
      setBusy('')
      setReportAfter(null)
    }
  }, [running])

  const reportLines = useMemo(() => {
    if (reportAfter === null) {
      return []
    }
    const commandIndex = entries.findIndex(
      (entry) => entry.seq > reportAfter && entry.message.trim() === `> ${SUMMARY_COMMAND}`,
    )
    if (commandIndex < 0) {
      return []
    }
    const response = entries
      .slice(commandIndex + 1)
      .filter((entry) => !entry.message.trim().startsWith('> '))
    return response
      .filter(
        (entry) =>
          entry.message.includes('|') ||
          REPORT_TITLE.test(entry.message) ||
          entry.mod.toLowerCase().includes('console'),
      )
      .map((entry) => entry.message)
  }, [entries, reportAfter])

  useEffect(() => {
    let active = true
    PerformanceReport(reportLines).then(
      (next) => {
        if (active) {
          setRows(next ?? [])
        }
      },
      (error: unknown) => {
        if (active) {
          setRows([])
        }
        reportUnexpected(error)
      },
    )
    return () => {
      active = false
    }
  }, [reportLines])

  const sortBy = (column: SortColumn) => {
    setSort((current) => ({
      column,
      direction: current.column === column && current.direction === 'asc' ? 'desc' : 'asc',
    }))
  }

  const startMeasuring = () => {
    if (!running || busy !== '' || measuring) {
      return
    }
    setBusy('start')
    send(game, ENABLE_COMMAND).then((sent) => {
      if (sent) {
        setMeasuring(true)
      }
      setBusy('')
    })
  }

  const showReport = () => {
    if (!running || busy !== '' || !measuring) {
      return
    }
    setRows([])
    setReportAfter(entries.at(-1)?.seq ?? 0)
    setBusy('report')
    send(game, SUMMARY_COMMAND).then((sent) => {
      if (!sent) {
        setReportAfter(null)
      }
      setBusy('')
    })
  }

  const copyReport = () => {
    Clipboard.SetText(reportLines.join('\n')).then(
      () => useToasts.getState().push({ kind: 'success', title: t`Report copied` }),
      reportUnexpected,
    )
  }

  return (
    <Paper
      variant="outlined"
      sx={{
        mx: 2,
        mb: 1,
        flexShrink: 0,
        overflow: 'hidden',
        bgcolor: 'rgba(0,0,0,0.24)',
        borderColor: 'rgba(255,255,255,0.14)',
      }}
    >
      <PanelHeader
        running={running}
        busy={busy}
        measuring={measuring}
        hasReport={reportLines.length > 0}
        onStart={startMeasuring}
        onReport={showReport}
        onCopy={copyReport}
      />
      <ReportBody rows={rows} reportLines={reportLines} sort={sort} onSort={sortBy} />
    </Paper>
  )
}
