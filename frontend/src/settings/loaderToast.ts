import { msg } from '@lingui/core/macro'
import type { Status } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loader/models.ts'
import { Status as LoaderStatus } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/loadersvc/service.ts'
import { SetLoaderToastAt } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { gameInfo } from '../games/info.ts'
import { i18n } from '../i18n/index.ts'
import { useLoader } from '../loader/store.ts'
import { useToasts } from '../toasts/store.ts'
import { useSettings } from './store.ts'

const ms = 1000
const dayMs = 24 * 60 * 60 * ms

export function loaderToastShownToday(at: string | undefined, now = Date.now()): boolean {
  if (!at) {
    return false
  }
  const then = Date.parse(at)
  return !Number.isNaN(then) && now - then < dayMs
}

// A loader that lives in each profile is updated in every profile on it, so the toast says so.
export function loaderUpdateText(name: string, status: Status): { title: string; body?: string } {
  const title = i18n._(msg`${name} ${{ latest: status.latest }} is out`)
  if (!status.perProfile) {
    return { title }
  }
  return {
    title,
    body: i18n._(
      msg`Your profiles have ${{ version: status.version }}. Update replaces each profile's copy.`,
    ),
  }
}

// The day's toast is kept per game and loader, since the games Mortar checks at start each have their own.
function maybeToastLoader(game: string, status: Status) {
  const loaderId = gameInfo(game)?.loaderId ?? ''
  const key = `${game}/${loaderId}`
  const { loaderPrefs } = useSettings.getState()
  if (loaderToastShownToday(loaderPrefs?.[key]?.toastAt)) {
    return
  }
  if (!(status.updateAvailable && status.latest)) {
    return
  }
  // The day counts as used only once a toast is shown, so an offline start does not silence it.
  const at = new Date().toISOString()
  useSettings.setState({
    loaderPrefs: { ...loaderPrefs, [key]: { ...loaderPrefs?.[key], toastAt: at } },
  })
  SetLoaderToastAt(game, loaderId, at).catch(() => undefined)
  useToasts.getState().push({
    kind: 'info',
    ...loaderUpdateText(gameInfo(game)?.loader ?? '', status),
    action: {
      label: i18n._(msg`Update`),
      run: () => useLoader.getState().install(game),
    },
  })
}

/**
 * Offers a newer loader for one game. It reads the status itself: the loader store holds the game the open screen shows,
 * and a check of every game at startup would leave that screen showing the last one checked.
 */
export async function checkLoaderUpdate(game: string) {
  const status = await LoaderStatus(game, '').catch(() => null)
  if (status) {
    maybeToastLoader(game, status)
  }
}
