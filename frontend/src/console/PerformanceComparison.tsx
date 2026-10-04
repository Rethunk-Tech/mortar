import { useLingui } from '@lingui/react/macro'
import { MenuItem, Select, Table, TableBody, TableCell, TableHead, TableRow } from '@mui/material'
import { useMemo } from 'react'
import type {
  PerformanceRow,
  SavedReport,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { formatTiming } from './formatTiming.ts'

const PERCENT = 100

export function CompareTable({ before, now }: { before: PerformanceRow[]; now: PerformanceRow[] }) {
  const { t, i18n } = useLingui()
  const rows = useMemo(() => {
    const byName = new Map<string, { before?: PerformanceRow; now?: PerformanceRow }>()
    for (const row of before) {
      byName.set(row.name, { before: row })
    }
    for (const row of now) {
      byName.set(row.name, { ...byName.get(row.name), now: row })
    }
    return [...byName.entries()]
      .map(([name, value]) => {
        const beforeMs = value.before?.averageMs ?? null
        const nowMs = value.now?.averageMs ?? null
        return {
          name,
          beforeMs,
          nowMs,
          change: beforeMs === null || nowMs === null ? null : nowMs - beforeMs,
          percent:
            beforeMs === null || nowMs === null || beforeMs === 0
              ? null
              : ((nowMs - beforeMs) / beforeMs) * PERCENT,
        }
      })
      .sort(
        (a, b) =>
          (b.change ?? (b.nowMs === null ? Number.NEGATIVE_INFINITY : Number.POSITIVE_INFINITY)) -
          (a.change ?? (a.nowMs === null ? Number.NEGATIVE_INFINITY : Number.POSITIVE_INFINITY)),
      )
  }, [before, now])
  const changeColor = (change: number | null) => {
    if (change === null) {
      return 'text.secondary'
    }
    return change > 0 ? 'warning.main' : 'success.main'
  }
  return (
    <Table size="small" stickyHeader={true} aria-label={t`Performance comparison`}>
      <TableHead>
        <TableRow>
          <TableCell>{t`Mod`}</TableCell>
          <TableCell align="right">{t`Before ms`}</TableCell>
          <TableCell align="right">{t`Now ms`}</TableCell>
          <TableCell align="right">{t`Change`}</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.name} hover={true}>
            <TableCell>{row.name}</TableCell>
            <TableCell align="right">
              {row.beforeMs === null ? t`new` : formatTiming(row.beforeMs, i18n.locale)}
            </TableCell>
            <TableCell align="right">
              {row.nowMs === null ? t`gone` : formatTiming(row.nowMs, i18n.locale)}
            </TableCell>
            <TableCell
              align="right"
              sx={{
                color: changeColor(row.change),
              }}
            >
              {row.change === null
                ? '—'
                : `${row.change >= 0 ? '+' : ''}${formatTiming(row.change, i18n.locale)} ms (${
                    row.percent === null
                      ? '—'
                      : `${row.percent >= 0 ? '+' : ''}${formatTiming(row.percent, i18n.locale)}%`
                  })`}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function ReportSelect({
  label,
  value,
  reports,
  onChange,
}: {
  label: string
  value: string
  reports: SavedReport[]
  onChange: (value: string) => void
}) {
  return (
    <Select
      size="small"
      value={value}
      displayEmpty={true}
      onChange={(event) => onChange(event.target.value)}
      slotProps={{ input: { 'aria-label': label } }}
      sx={{ minWidth: 145, fontSize: 12 }}
    >
      <MenuItem value="">{label}</MenuItem>
      {reports.map((report) => (
        <MenuItem key={report.id} value={report.id}>
          {formatWhen(report.at, { withTime: true })}
        </MenuItem>
      ))}
    </Select>
  )
}
