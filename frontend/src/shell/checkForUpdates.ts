import { msg, plural } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { useUpdates } from '../mods/updates.ts'
import { useNav } from '../nav/store.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { errorDetails, errorMessage } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// F5 and the app menu's Check for updates: the open profile's mods (bypassing cached answers) and Mortar itself,
// reported in one toast.
export async function checkForUpdates() {
  const toasts = useToasts.getState()
  toasts.push({ kind: 'info', title: i18n._(msg`Checking for updates…`) })
  const [mods, mortar] = await Promise.allSettled([
    useUpdates.getState().checkNow(),
    useMortarUpdate.getState().check(),
  ])
  if (mods.status === 'rejected') {
    toasts.push({
      kind: 'error',
      title: i18n._(msg`Mod update check failed`),
      body: errorMessage(mods.reason),
      detail: errorDetails(mods.reason),
    })
  }
  const { phase } = useMortarUpdate.getState()
  const mortarNews = phase === 'available' || phase === 'ready'
  const parts: string[] = []
  const found = mods.status === 'fulfilled' ? mods.value : null
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
  if (parts.length === 0) {
    return
  }
  toasts.push({
    kind: (found ?? 0) > 0 || mortarNews ? 'success' : 'info',
    title: parts.join(' · '),
    ...(mortarNews
      ? {
          action: {
            label: i18n._(msg`View`),
            run: () => useNav.getState().openSettings('updates'),
          },
        }
      : {}),
  })
}
