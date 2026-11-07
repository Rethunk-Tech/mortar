import { msg, plural } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { create } from 'zustand'
import type { SweepReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { MACOS, STEAM } from '../brand/productNames.ts'
import { useConsole } from '../console/store.ts'
import { i18n } from '../i18n/index.ts'
import { useSettings } from '../settings/store.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { applyLive, clearLiveUnlessRunning } from './live.ts'
import { useLaunchView } from './progress.ts'
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
  const version = report.gameVersion || report.loaderVersion
  const name = report.gameName || report.game
  useToasts.getState().push({
    kind: 'warning',
    title: i18n._(
      msg`${name} updated to ${version}: ${plural({ count: n }, { one: '# profile needs attention', other: '# profiles need attention' })}`,
    ),
    action: {
      label: i18n._(msg`Review`),
      run: () => useSweepUi.getState().review(),
    },
  })
}

export function initLaunch() {
  Events.On('launch:state', (event) => {
    useLaunch.getState().apply(event.data)
    clearLiveUnlessRunning(event.data)
  })
  Events.On('launch:live', (event) => applyLive(event.data))
  Events.On('launch:line', (event) => useConsole.getState().add(event.data))
  Events.On('launch:progress', (event) => useLaunchView.getState().onProgress(event.data))
  Events.On('launch:backup-warning', (event) => {
    const data = event.data as { error?: string }
    const details = errorDetails(data.error)
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(msg`Could not back up saves before Play`),
      body: errorMessage(data.error),
      ...(details === '' ? {} : { detail: details }),
    })
  })
  Events.On('launch:steam-option', (event) => {
    const data = event.data as { option?: string }
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(
        msg`Set this ${{ launcher: STEAM }} launch option so the mod loader starts on ${{ os: MACOS }}`,
      ),
      body: data.option ?? '',
    })
  })
  Events.On('avscan:failed', (event) => {
    const data = event.data as { error?: string }
    useToasts.getState().push({
      kind: 'warning',
      title: i18n._(msg`The antivirus scan could not finish, so a mod was installed unscanned`),
      body: data.error ?? '',
    })
  })
  Events.On('catalog:stale', (event) => {
    const data = event.data as { issued?: string }
    const issued = data.issued ? new Date(data.issued) : null
    const date =
      issued && !Number.isNaN(issued.getTime()) ? issued.toLocaleDateString(i18n.locale) : ''
    useToasts.getState().push({
      kind: 'info',
      topic: 'catalog-stale',
      title: i18n._(msg`The game catalog could not be refreshed`),
      body: i18n._(
        msg`Mortar keeps using the copy signed on ${date}. Check your connection and try again.`,
      ),
      action: {
        label: i18n._(msg`Try again`),
        run: () => Events.Emit('catalog:retry', ''),
      },
    })
  })
  Events.On('launch:settings-restore-warning', (event) => {
    const data = event.data as { error?: string }
    toastError(i18n._(msg`Could not restore profile game settings`), data.error)
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
