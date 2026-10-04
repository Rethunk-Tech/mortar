import { useLingui } from '@lingui/react/macro'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { useStoredState } from '../shell/useStoredState.ts'
import { MeasuredPanel, PerformanceEmpty } from './PerformancePanelParts.tsx'
import {
  copyReportLines,
  type SortColumn,
  type SortDirection,
  usePanelControls,
  useParsedReport,
  usePerformanceRunning,
  useReportLines,
  useSavedReports,
} from './usePerformancePanel.ts'

const SORT_COLUMNS: readonly unknown[] = ['name', 'averageMs', 'peakMs', 'calls']

const isStoredSort = (value: unknown): value is { column: SortColumn; direction: SortDirection } =>
  typeof value === 'object' &&
  value !== null &&
  'column' in value &&
  'direction' in value &&
  SORT_COLUMNS.includes(value.column) &&
  (value.direction === 'asc' || value.direction === 'desc')

export function PerformancePanel({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const running = usePerformanceRunning(game)
  const [measuring, setMeasuring] = useState(false)
  const [reportAfter, setReportAfter] = useState<number | null>(null)
  const [savedReports, setSavedReports] = useSavedReports(game, openId)
  const [compareId, setCompareId] = useState('')
  const [beforeId, setBeforeId] = useState('')
  const [afterId, setAfterId] = useState('')
  const [sort, setSort] = useStoredState<{ column: SortColumn; direction: SortDirection }>(
    'mortar.performanceSort',
    { column: 'peakMs', direction: 'desc' },
    isStoredSort,
  )
  const reportLines = useReportLines(reportAfter)
  const [rows, setRows] = useParsedReport(reportLines, game, openId, setSavedReports)
  const { busy, startMeasuring, showReport } = usePanelControls({
    game,
    running,
    measuring,
    setMeasuring,
    setRows,
    setReportAfter,
  })
  const comparisonBefore = compareId
    ? (savedReports.find((report) => report.id === compareId)?.rows ?? [])
    : (savedReports.find((report) => report.id === beforeId)?.rows ?? [])
  const comparisonNow = compareId
    ? rows
    : (savedReports.find((report) => report.id === afterId)?.rows ?? [])
  const comparing = (compareId !== '' && rows.length > 0) || (beforeId !== '' && afterId !== '')
  if (reportLines.length === 0 && !measuring && !comparing) {
    return (
      <PerformanceEmpty
        running={running}
        busy={busy !== ''}
        onStart={startMeasuring}
        reports={savedReports}
        onCompare={() => {
          setBeforeId(savedReports[1]?.id ?? '')
          setAfterId(savedReports[0]?.id ?? '')
        }}
      />
    )
  }
  return (
    <MeasuredPanel
      header={{
        running,
        busy,
        measuring,
        hasReport: reportLines.length > 0,
        onStart: startMeasuring,
        onReport: showReport,
        onCopy: () => copyReportLines(reportLines, t`Report copied`),
        reports: savedReports,
        compareId,
        onCompare: setCompareId,
        onClearCompare: () => {
          setCompareId('')
          setBeforeId('')
          setAfterId('')
        },
        compareSides: rows.length === 0,
        beforeId,
        afterId,
        onBefore: setBeforeId,
        onAfter: setAfterId,
      }}
      comparing={comparing}
      comparisonBefore={comparisonBefore}
      comparisonNow={comparisonNow}
      rows={rows}
      reportLines={reportLines}
      sort={sort}
      onSort={(column) =>
        setSort({
          column,
          direction: sort.column === column && sort.direction === 'asc' ? 'desc' : 'asc',
        })
      }
    />
  )
}
