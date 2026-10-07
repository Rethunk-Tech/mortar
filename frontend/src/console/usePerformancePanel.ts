
import { useEffect, useMemo, useState } from 'react'
import type {
  PerformanceRow,
  SavedReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  PerformanceReport,
  PerformanceReports,
  SavePerformanceReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { copyText } from '../share/copyText.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { canSendTo, useConsole } from './store.ts'

const ENABLE_COMMAND = 'performance enable'
const SUMMARY_COMMAND = 'performance summary'
const REPORT_TITLE = /summary|performance counter/i
const MAX_SAVED_REPORTS = 20

type SortColumn = 'name' | 'averageMs' | 'peakMs' | 'calls'
type SortDirection = 'asc' | 'desc'
type PanelBusy = 'start' | 'report' | ''

function rememberSaved(saved: SavedReport) {
  return (current: SavedReport[]) =>
    [saved, ...current.filter((item) => item.id !== saved.id)].slice(0, MAX_SAVED_REPORTS)
}

function usePerformanceRunning(game: string) {
  const viewingRun = useConsole((s) => s.viewingRun)
  const status = useLaunch((s) => s.status)
  const openId = useProfiles((s) => s.openId)
  return viewingRun === '' && canSendTo(status, game, openId)
}

function useSavedReports(game: string, openId: string) {
  const [savedReports, setSavedReports] = useState<SavedReport[]>([])
  useEffect(() => {
    if (openId === '') {
      return
    }
    PerformanceReports(game, openId).then(
      (reports) => setSavedReports(reports ?? []),
      reportUnexpected,
    )
  }, [game, openId])
  return [savedReports, setSavedReports] as const
}

function useReportLines(reportAfter: number | null) {
  const entries = useConsole((s) => s.entries)
  return useMemo(() => {
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
}

function useParsedReport(
  reportLines: string[],
  game: string,
  openId: string,
  setSavedReports: (update: (current: SavedReport[]) => SavedReport[]) => void,
) {
  const [rows, setRows] = useState<PerformanceRow[]>([])
  useEffect(() => {
    let active = true
    const run = async () => {
      try {
        const next = await PerformanceReport(reportLines)
        if (!active) {
          return
        }
        const parsed = next ?? []
        setRows(parsed)
        if (parsed.length === 0) {
          return
        }
        try {
          const saved = await SavePerformanceReport(game, openId, parsed)
          if (active) {
            setSavedReports(rememberSaved(saved))
          }
        } catch (error) {
          reportUnexpected(error)
        }
      } catch (error) {
        if (active) {
          setRows([])
        }
        reportUnexpected(error)
      }
    }
    run().then(
      () => undefined,
      () => undefined,
    )
    return () => {
      active = false
    }
  }, [reportLines, game, openId, setSavedReports])
  return [rows, setRows] as const
}

function copyReportLines(reportLines: string[], copiedTitle: string) {
  void copyText(reportLines.join('\n'), copiedTitle)
}

function usePanelControls(opts: {
  game: string
  running: boolean
  measuring: boolean
  setMeasuring: (on: boolean) => void
  setRows: (rows: PerformanceRow[]) => void
  setReportAfter: (n: number | null) => void
}) {
  const { game, running, measuring, setMeasuring, setRows, setReportAfter } = opts
  const send = useConsole((s) => s.send)
  const entries = useConsole((s) => s.entries)
  const [busy, setBusy] = useState<PanelBusy>('')
  useEffect(() => {
    if (!running) {
      setMeasuring(false)
      setBusy('')
      setReportAfter(null)
    }
  }, [running, setMeasuring, setReportAfter])
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
  return { busy, startMeasuring, showReport }
}

export type { PanelBusy, SortColumn, SortDirection }
export {
  copyReportLines,
  usePanelControls,
  useParsedReport,
  usePerformanceRunning,
  useReportLines,
  useSavedReports,
}
