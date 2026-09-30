import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { useLoader } from '../loader/store.ts'
import { useToasts } from '../toasts/store.ts'

const storageKey = 'mortar-smapi-check-at'
const ms = 1000
const dayMs = 24 * 60 * 60 * ms

function checkedToday(): boolean {
  const raw = localStorage.getItem(storageKey)
  if (!raw) {
    return false
  }
  const then = Date.parse(raw)
  return !Number.isNaN(then) && Date.now() - then < dayMs
}

export function maybeToastSmapi(game: string) {
  if (checkedToday()) {
    return
  }
  localStorage.setItem(storageKey, new Date().toISOString())
  const { status } = useLoader.getState()
  if (!(status?.updateAvailable && status.latest)) {
    return
  }
  useToasts.getState().push({
    kind: 'info',
    title: i18n._(msg`SMAPI ${status.latest} is out`),
    action: {
      label: i18n._(msg`Update`),
      run: () => useLoader.getState().install(game),
    },
  })
}
