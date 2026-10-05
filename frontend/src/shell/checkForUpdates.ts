import { msg, plural } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { useUpdates } from '../mods/updates.ts'
import { useNav } from '../nav/store.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, toastError } from '../toasts/report.ts'
import { type ToastInput, useToasts } from '../toasts/store.ts'
import { updatesOfflineReason } from './offlineText.ts'

function failReason(
  mods: PromiseSettledResult<unknown>,
  mortar: PromiseSettledResult<unknown>,
): unknown {
  if (mods.status === 'rejected') {
    return mods.reason
  }
  if (mortar.status === 'rejected') {
    return mortar.reason
  }
  return null
}

function resultKind(
  found: number | null,
  mortarNews: boolean,
  fail: unknown,
): 'success' | 'warning' | 'info' {
  if ((found ?? 0) > 0 || mortarNews) {
    return 'success'
  }
  if (fail !== null) {
    return 'warning'
  }
  return 'info'
}

// F5 and the app menu's Check for updates: the open profile's mods (bypassing cached answers) and Mortar itself,
// reported in one toast.
export async function checkForUpdates() {
  const toasts = useToasts.getState()
  const offline = updatesOfflineReason()
  if (offline !== '') {
    toasts.push({ kind: 'warning', title: i18n._(msg`Update check skipped`), body: offline })
    return
  }
  const checking = toasts.push({ kind: 'info', title: i18n._(msg`Checking for updates…`) })
  const [mods, mortar] = await Promise.allSettled([
    useUpdates.getState().checkNow(),
    useMortarUpdate.getState().check(),
  ])
  const { phase } = useMortarUpdate.getState()
  const mortarNews = phase === 'available' || phase === 'ready'
  const parts: string[] = []
  const found = mods.status === 'fulfilled' && typeof mods.value === 'number' ? mods.value : null
  if (found !== null) {
    parts.push(
      found > 0
        ? plural(found, { one: '# mod update', other: '# mod updates' })
        : i18n._(msg`Mods are up to date`),
    )
  }
  if (mortar.status === 'fulfilled' && phase !== 'error') {
    parts.push(mortarNews ? i18n._(msg`a new Mortar is ready`) : i18n._(msg`Mortar is up to date`))
  }
  const fail = failReason(mods, mortar)
  toasts.dismiss(checking)
  if (parts.length === 0) {
    if (fail !== null) {
      toastError(i18n._(msg`Update check failed`), fail, { retry: checkForUpdates })
    }
    return
  }
  const toast: ToastInput = {
    kind: resultKind(found, mortarNews, fail),
    title: parts.join(' · '),
  }
  if (fail !== null) {
    toast.body = errorMessage(fail)
    toast.detail = errorDetails(fail)
  } else if (mortarNews) {
    toast.action = {
      label: i18n._(msg`View`),
      run: () => useNav.getState().openSettings('updates'),
    }
  }
  toasts.push(toast)
}
