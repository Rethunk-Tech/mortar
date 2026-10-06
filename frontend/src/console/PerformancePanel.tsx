import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { type ReactNode, useState } from 'react'
import type {
  FrameSummary,
  PerformanceRow,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { useStoredState } from '../shell/useStoredState.ts'
import { formatTiming } from './formatTiming.ts'
import { MeasuredPanel, PerformanceEmpty } from './PerformancePanelParts.tsx'
import { usePerfQuery } from './startupHooks.ts'
import { useBridgePerf } from './useBridgePerf.ts'
import {
  copyReportLines,
  type PanelBusy,
  type SortColumn,
  type SortDirection,
  usePanelControls,
  useParsedReport,
  usePerformanceRunning,
  useReportLines,
  useSavedReports,
} from './usePerformancePanel.ts'

const MIB = 1_048_576
const SORT_COLUMNS: readonly unknown[] = ['name', 'averageMs', 'peakMs', 'calls']

const isStoredSort = (value: unknown): value is { column: SortColumn; direction: SortDirection } =>
  typeof value === 'object' &&
  value !== null &&
  'column' in value &&
  'direction' in value &&
  SORT_COLUMNS.includes(value.column) &&
  (value.direction === 'asc' || value.direction === 'desc')

interface Source {
  running: boolean
  busy: PanelBusy
  measuring: boolean
  rows: PerformanceRow[]
  reportLines: string[]
  startMeasuring: () => void
  showReport: () => void
  /** Shown above the report: the companion's frame summary and what its rows count. */
  summary?: ReactNode
  /** Replaces the empty state's text. */
  hint?: string
}

function PerformancePanel({ game }: { game: string }) {
  return usePerfQuery() ? <BridgePanel game={game} /> : <ConsolePanel game={game} />
}

// SMAPI answers through its console: the report is the command's text output, parsed.
function ConsolePanel({ game }: { game: string }) {
  const openId = useProfiles((s) => s.openId)
  const running = usePerformanceRunning(game)
  const [measuring, setMeasuring] = useState(false)
  const [reportAfter, setReportAfter] = useState<number | null>(null)
  const saved = useSavedReports(game, openId)
  const reportLines = useReportLines(reportAfter)
  const [rows, setRows] = useParsedReport(reportLines, game, openId, saved[1])
  const { busy, startMeasuring, showReport } = usePanelControls({
    game,
    running,
    measuring,
    setMeasuring,
    setRows,
    setReportAfter,
  })
  return (
    <PanelView
      source={{ running, busy, measuring, rows, reportLines, startMeasuring, showReport }}
      saved={saved}
    />
  )
}

function BridgePanel({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const running = usePerformanceRunning(game)
  const saved = useSavedReports(game, openId)
  const perf = useBridgePerf({ game, openId, running, setSavedReports: saved[1] })
  return (
    <PanelView
      source={{
        ...perf,
        running,
        summary: <FrameLine frame={perf.frame} />,
        hint: perf.unmeasured
          ? t`This launch was not measured. Use Measure next launch under Startup, play again, then start measuring here.`
          : t`Use Measure next launch under Startup and play; then start measuring here to time each plugin per frame.`,
      }}
      saved={saved}
    />
  )
}

function PanelView({
  source,
  saved,
}: {
  source: Source
  saved: ReturnType<typeof useSavedReports>
}) {
  const { t } = useLingui()
  const { running, busy, measuring, rows, reportLines, startMeasuring, showReport } = source
  const [savedReports] = saved
  const [compareId, setCompareId] = useState('')
  const [beforeId, setBeforeId] = useState('')
  const [afterId, setAfterId] = useState('')
  const [sort, setSort] = useStoredState<{ column: SortColumn; direction: SortDirection }>(
    'mortar.performanceSort',
    { column: 'peakMs', direction: 'desc' },
    isStoredSort,
  )
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
        hint={source.hint}
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
      summary={comparing ? null : source.summary}
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

function FrameLine({ frame }: { frame: FrameSummary | null }) {
  const { t, i18n } = useLingui()
  const ms = (v: number) => formatTiming(v, i18n.locale)
  const mib = (bytes: number) => Math.round(bytes / MIB).toLocaleString(i18n.locale)
  return (
    <Box
      sx={{
        px: 1.5,
        pb: 1,
        fontSize: 12,
        color: 'text.secondary',
        display: 'flex',
        flexDirection: 'column',
        gap: 0.25,
      }}
    >
      {frame ? (
        <span>
          {t`${Math.round(frame.fps)} fps over ${Math.round(frame.seconds)} s. Frame ms: average ${ms(frame.avgMs)}, 95th ${ms(frame.p95Ms)}, 99th ${ms(frame.p99Ms)}, worst ${ms(frame.maxMs)}. Mono heap ${mib(frame.monoUsed)} of ${mib(frame.monoHeap)} MiB, ${frame.gcCollections} garbage collections.`}
        </span>
      ) : null}
      <span>
        {t`Each plugin's main-thread time per frame: its Harmony prefixes, postfixes and finalizers and its own Update, LateUpdate and FixedUpdate, less any timed call inside them. Code a transpiler rewrote and work on other threads are not counted.`}
      </span>
    </Box>
  )
}

export { PerformancePanel }
