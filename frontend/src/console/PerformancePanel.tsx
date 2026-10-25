import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { type ReactNode, useState } from 'react'
import type {
  FrameSummary,
  PerformanceRow,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { BEPINEX, HARMONY } from '../brand/productNames.ts'
import { useProfileLoader, useProfiles } from '../profiles/store.ts'
import { useStoredState } from '../shell/useStoredState.ts'
import { space } from '../theme/density.ts'
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
const SORT_COLUMNS: readonly unknown[] = ['name', 'averageMs', 'p95Ms', 'peakMs', 'share', 'calls']

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
  /** When measuring started (ms), for the elapsed time; absent where it is not known. */
  since?: number
  /** Why the game stopped measuring, when it did. */
  reason?: string
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
  const smapi = useProfileLoader()?.id === 'smapi'
  let hint = t`Start the game, then start measuring here to time each plugin per frame.`
  if (smapi) {
    hint = t`Start the game, then start measuring here to time each mod's event handlers per frame.`
  } else if (perf.reason) {
    hint = t`Measuring stopped: ${{ reason: perf.reason }}`
  } else if (perf.unmeasured) {
    hint = t`This ${{ loader: BEPINEX }} Bridge measures in game only on a measured launch. Use Measure next launch under Startup and play again.`
  }
  return (
    <PanelView
      source={{
        ...perf,
        running,
        summary: <FrameLine frame={perf.frame} smapi={smapi} />,
        hint,
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
        since: source.since,
        reason: source.reason,
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

function FrameLine({ frame, smapi }: { frame: FrameSummary | null; smapi: boolean }) {
  const { t, i18n } = useLingui()
  const ms = (v: number) => formatTiming(v, i18n.locale)
  const mib = (bytes: number) => Math.round(bytes / MIB).toLocaleString(i18n.locale)
  return (
    <Box
      sx={{
        px: space.pad,
        pb: space.gap,
        fontSize: 'small.fontSize',
        color: 'text.secondary',
        display: 'flex',
        flexDirection: 'column',
        gap: 0.25,
      }}
    >
      {frame ? (
        <span>
          {t`${{ fps: Math.round(frame.fps) }} fps over ${{ seconds: Math.round(frame.seconds) }} s. Frame ms: average ${{ avg: ms(frame.avgMs) }}, 95th ${{ p95: ms(frame.p95Ms) }}, 99th ${{ p99: ms(frame.p99Ms) }}, worst ${{ max: ms(frame.maxMs) }}. Managed heap ${{ monoUsed: mib(frame.monoUsed) }} of ${{ monoHeap: mib(frame.monoHeap) }} MiB, ${{ gcCollections: frame.gcCollections }} garbage collections.`}
        </span>
      ) : null}
      {frame && frame.modsMs > 0 ? (
        <span>
          {t`Baseline: the mods' timed code is ${{ mods: ms(frame.modsMs) }} ms of the ${{ avg: ms(frame.avgMs) }} ms average frame; the game and loader alone take about ${{ withoutMods: ms(frame.withoutModsMs) }} ms.`}
        </span>
      ) : null}
      <span>
        {smapi
          ? t`Each mod's time in its per-frame event handlers (update, render and one-second ticks), summed per frame, averaged over every frame. The 95th percentile and peak are per frame. ${{ app: HARMONY }} patches and work outside events are in the baseline, not in a mod's row.`
          : t`Each plugin's main-thread time per frame: its ${{ app: HARMONY }} prefixes, postfixes and finalizers and its own Update, LateUpdate and FixedUpdate, less any timed call inside them. Code a transpiler rewrote and work on other threads are not counted.`}
      </span>
    </Box>
  )
}

export { PerformancePanel }
