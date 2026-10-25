import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import type {
  FrameSummary,
  PerformanceRow,
  SavedReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  MeasureInGame,
  PeekInGame,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { reportError } from '../toasts/report.ts'
import { formatTiming } from './formatTiming.ts'
import { usePerfLive } from './perfLive.ts'
import type { PanelBusy } from './usePerformancePanel.ts'

const PERCENT = 100

const MAX_SAVED_REPORTS = 20
// How often a running measurement is read for the live table.
const LIVE_EVERY_MS = 2000

// While measuring, the table follows the game: a read every few seconds, saved only when the player asks.
function useLivePeek(
  game: string,
  openId: string,
  measuring: boolean,
  set: {
    setRows: (rows: PerformanceRow[]) => void
    setFrame: (frame: FrameSummary | null) => void
  },
) {
  const { setRows, setFrame } = set
  const { t } = useLingui()
  useEffect(() => {
    if (!measuring) {
      return
    }
    let alive = true
    const peek = () => {
      PeekInGame(game, openId)
        .then((live) => {
          if (!alive) {
            return
          }
          if (!live.measured) {
            usePerfLive.setState({
              measuring: false,
              reason: live.reason || t`The game stopped measuring.`,
            })
            return
          }
          const got = live.rows ?? []
          setRows(got)
          setFrame(live.frame ?? null)
          const top = got.reduce<PerformanceRow | null>(
            (best, r) => (best === null || r.averageMs > best.averageMs ? r : best),
            null,
          )
          usePerfLive.setState({ top: top?.name ?? '' })
        })
        .catch(() => undefined)
    }
    peek()
    const timer = setInterval(peek, LIVE_EVERY_MS)
    return () => {
      alive = false
      clearInterval(timer)
    }
  }, [measuring, game, openId, t, setRows, setFrame])
}

/** The in-game measurement a loader's companion answers for as data (BepInEx): start it, then read a summary, which
 * the backend saves with the profile's reports. Only a launch measured from Startup has one. */
export function useBridgePerf(opts: {
  game: string
  openId: string
  running: boolean
  setSavedReports: (update: (current: SavedReport[]) => SavedReport[]) => void
}) {
  const { game, openId, running, setSavedReports } = opts
  const { t, i18n } = useLingui()
  const [busy, setBusy] = useState<PanelBusy>('')
  const measuring = usePerfLive((s) => s.measuring)
  const since = usePerfLive((s) => s.since)
  const reason = usePerfLive((s) => s.reason)
  const [unmeasured, setUnmeasured] = useState(false)
  const [rows, setRows] = useState<PerformanceRow[]>([])
  const [frame, setFrame] = useState<FrameSummary | null>(null)
  useEffect(() => {
    if (!running) {
      usePerfLive.setState({ measuring: false, top: '' })
      setUnmeasured(false)
      setBusy('')
    }
  }, [running])
  useLivePeek(game, openId, measuring, { setRows, setFrame })
  const startMeasuring = () => {
    if (!running || busy !== '' || measuring) {
      return
    }
    setBusy('start')
    MeasureInGame(game, openId, true)
      .then((r) => {
        setUnmeasured(!r.measured)
        usePerfLive.setState({
          measuring: r.measured,
          since: Date.now(),
          reason: r.measured ? '' : (r.reason ?? ''),
          top: '',
        })
      })
      .catch(reportError(t`Could not start measuring`, startMeasuring))
      .finally(() => setBusy(''))
  }
  const showReport = () => {
    if (!running || busy !== '' || !measuring) {
      return
    }
    setBusy('report')
    MeasureInGame(game, openId, false)
      .then((r) => {
        const saved = r.report
        if (!saved) {
          return
        }
        setRows(saved.rows ?? [])
        setFrame(saved.frame ?? null)
        setSavedReports((current) =>
          [saved, ...current.filter((item) => item.id !== saved.id)].slice(0, MAX_SAVED_REPORTS),
        )
      })
      .catch(reportError(t`Could not read the measurement`, showReport))
      .finally(() => setBusy(''))
  }
  const reportLines =
    rows.length === 0
      ? []
      : [
          'Mod | Average ms | 95th ms | Peak ms | % of frame | Calls per frame',
          ...rows.map((r) =>
            [
              r.name,
              r.averageMs,
              r.p95Ms ?? 0,
              r.peakMs,
              `${((r.share ?? 0) * PERCENT).toFixed(1)}%`,
              r.calls,
            ]
              .map((v) => (typeof v === 'number' ? formatTiming(v, i18n.locale) : v))
              .join(' | '),
          ),
        ]
  return {
    busy,
    measuring,
    since,
    reason,
    unmeasured,
    rows,
    frame,
    reportLines,
    startMeasuring,
    showReport,
  }
}
