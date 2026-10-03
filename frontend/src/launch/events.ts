import { msg, plural } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { SweepReport } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useConsole } from '../console/store.ts'
import { i18n } from '../i18n/index.ts'
import { useSettings } from '../settings/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useLaunch } from './store.ts'

const useSweepUi = create<{
  report: SweepReport | null
  open: boolean
  present: (report: SweepReport) => void
  review: () => void
  close: () => void
}>((set) => ({
  report: null,
  open: false,
  present: (report) => set({ report }),
  review: () => set({ open: true }),
  close: () => set({ open: false }),
}))

function attentionCount(report: SweepReport): number {
  return (report.profiles ?? []).filter(
    (row) => (row.broken?.length ?? 0) > 0 || (row.missingDeps ?? 0) > 0,
  ).length
}

function toastSweep(report: SweepReport) {
  const n = attentionCount(report)
  if (n === 0) {
    return
  }
  useSweepUi.getState().present(report)
  const version = report.gameVersion || report.smapiVersion
  const name = report.gameName || report.game
  useToasts.getState().push({
    kind: 'warning',
    title: i18n._(
      msg`${name} updated to ${version} — ${plural(n, { one: '# profile needs attention', other: '# profiles need attention' })}`,
    ),
    action: {
      label: i18n._(msg`Review`),
      run: () => useSweepUi.getState().review(),
    },
  })
}

export function initLaunch() {
  Events.On('launch:state', (event) => useLaunch.getState().apply(event.data))
  Events.On('launch:line', (event) => useConsole.getState().add(event.data))
  Events.On('launch:backup-warning', (event) => {
    const data = event.data as { error?: string }
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(msg`Could not back up saves before Play`),
      body: data.error ?? '',
    })
  })
  Events.On('launch:settings-restore-warning', (event) => {
    const data = event.data as { error?: string }
    useToasts.getState().push({
      kind: 'error',
      title: i18n._(msg`Could not restore profile game settings`),
      body: data.error ?? '',
    })
  })
  Events.On('launch:crash', (event) => {
    if (useSettings.getState().notifyRunCrashed === false) {
      return
    }
    useLaunch.getState().setCrash(event.data)
  })
  Events.On('launch:sweep', (event) => toastSweep(event.data as SweepReport))
}

export { useSweepUi }
