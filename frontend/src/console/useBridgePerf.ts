import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import type {
  FrameSummary,
  PerformanceRow,
  SavedReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { MeasureInGame } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { reportError } from '../toasts/report.ts'
import { formatTiming } from './formatTiming.ts'
import type { PanelBusy } from './usePerformancePanel.ts'

const PERCENT = 100

const MAX_SAVED_REPORTS = 20

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
  const [measuring, setMeasuring] = useState(false)
  const [unmeasured, setUnmeasured] = useState(false)
  const [rows, setRows] = useState<PerformanceRow[]>([])
  const [frame, setFrame] = useState<FrameSummary | null>(null)
  useEffect(() => {
    if (!running) {
      setMeasuring(false)
      setUnmeasured(false)
      setBusy('')
    }
  }, [running])
  const startMeasuring = () => {
    if (!running || busy !== '' || measuring) {
      return
    }
    setBusy('start')
    MeasureInGame(game, openId, true)
      .then((r) => {
        setUnmeasured(!r.measured)
        setMeasuring(r.measured)
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
  return { busy, measuring, unmeasured, rows, frame, reportLines, startMeasuring, showReport }
}
