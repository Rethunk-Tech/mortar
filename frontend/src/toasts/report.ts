import { msg } from '@lingui/core/macro'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { openSettings, routeGame, useNav } from '../nav/store.ts'
import { errorDetails, errorKind as kindOf } from './errorKind.ts'

import { type ToastAction, useToasts } from './store.ts'

// Built on call, not at import: Lingui macros only run inside compiled code, and this module is
// imported nearly everywhere, tests included.
function sentence(kind: ReturnType<typeof kindOf>): string {
  switch (kind) {
    case 'not_found':
      return i18n._(msg`That item could not be found.`)
    case 'busy':
      return i18n._(msg`The game is already running.`)
    case 'network':
      return i18n._(msg`A network request failed.`)
    case 'permission':
      return i18n._(msg`Mortar does not have permission to do that.`)
    case 'disk_full':
      return i18n._(msg`The disk is full.`)
    case 'damaged':
      return i18n._(msg`That data could not be read.`)
    case 'invalid':
      return i18n._(msg`That request was not valid.`)
    case 'other_game':
      return i18n._(msg`That is for another game. Open that game and import it there.`)
    case 'outdated':
      return i18n._(msg`That was made by a newer Mortar. Update Mortar to open it.`)
    case 'external':
      return i18n._(
        msg`The author only allows this download on the source's own page. Download the file there and add it from your computer.`,
      )
    default:
      return i18n._(msg`Something went wrong`)
  }
}

/** Brings the console tab of the current (or originating) game forward; false when no game is open. */
function openLog(): boolean {
  const nav = useNav.getState()
  const game = routeGame(nav.route)
  if (game === null) {
    return false
  }
  if (nav.route.name !== 'game') {
    nav.openGame(game)
  }
  useTab.getState().setTab('console')
  return true
}

// Every error toast offers a next step: the caller's own action, Retry for a repeatable one, Settings for
// permission and storage failures, else the game's log, or the diagnostics when no game is open.
function nextStep(e: unknown, retry: (() => unknown) | undefined): ToastAction {
  const kind = kindOf(e)
  if (kind === 'permission' || kind === 'disk_full') {
    return { label: i18n._(msg`Open storage settings`), run: () => openSettings('storage') }
  }
  if (retry !== undefined) {
    return { label: i18n._(msg`Retry`), run: retry }
  }
  if (routeGame(useNav.getState().route) === null) {
    // No game, so no run log: Mortar's own log goes out with the diagnostics on About.
    return { label: i18n._(msg`Save diagnostics…`), run: () => openSettings('about') }
  }
  return { label: i18n._(msg`Open log`), run: openLog }
}

/** A failure shown in place: the plain sentence to read, the cause for its title tooltip. */
export interface InlineError {
  message: string
  details: string
}

/** Plain sentence for a kind. Untagged errors use the generic sentence; raw Go text stays in errorDetails. */
export function errorMessage(e: unknown): string {
  return sentence(kindOf(e))
}

export function inlineError(e: unknown, message = errorMessage(e)): InlineError {
  return { message, details: errorDetails(e) }
}

/** `detail` is a line appended after the error's own text; `action` replaces the default next step; `retry` repeats the failed action. */
export function toastError(
  title: string,
  e: unknown,
  extra: { action?: ToastAction; detail?: string; retry?: () => unknown } = {},
): void {
  const details = [errorDetails(e), extra.detail ?? ''].filter((part) => part !== '').join('\n')
  const action = extra.action ?? nextStep(e, extra.retry)
  const body = errorMessage(e)
  useToasts.getState().push({
    kind: 'error',
    title,
    // An untagged error's sentence is the generic title itself, which would only repeat it.
    ...(body === title ? {} : { body }),
    ...(details === '' ? {} : { detail: details }),
    action,
  })
}

export const reportError = (title: string, retry?: () => unknown) => (e: unknown) => {
  toastError(title, e, retry === undefined ? {} : { retry })
}

// The sink for a promise nobody awaits. Stores report their own failures, so this only fires for one they did not expect.
// A network failure is expected whenever a source is down, and the offline banner already says so.
export const reportUnexpected = (e: unknown) => {
  if (kindOf(e) === 'network') {
    return
  }
  toastError(i18n._(msg`Something went wrong`), e)
}
