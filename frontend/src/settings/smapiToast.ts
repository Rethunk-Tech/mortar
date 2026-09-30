import { msg } from '@lingui/core/macro'
import { SetSmapiToastAt } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { useLoader } from '../loader/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useSettings } from './store.ts'

const ms = 1000
const dayMs = 24 * 60 * 60 * ms

export function smapiToastShownToday(at: string | undefined, now = Date.now()): boolean {
  if (!at) {
    return false
  }
  const then = Date.parse(at)
  return !Number.isNaN(then) && now - then < dayMs
}

export function maybeToastSmapi(game: string) {
  if (smapiToastShownToday(useSettings.getState().smapiToastAt)) {
    return
  }
  const at = new Date().toISOString()
  useSettings.setState({ smapiToastAt: at })
  SetSmapiToastAt(at).catch(() => undefined)
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
