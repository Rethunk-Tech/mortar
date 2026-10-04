import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
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
    default:
      return i18n._(msg`Something went wrong.`)
  }
}

/** Plain sentence for a kind. Untagged errors use the generic sentence; raw Go text stays in errorDetails. */
export function errorMessage(e: unknown): string {
  return sentence(kindOf(e))
}

/** A failure shown in place: the plain sentence to read, the cause for its title tooltip. */
export interface InlineError {
  message: string
  details: string
}

export function inlineError(e: unknown, message = errorMessage(e)): InlineError {
  return { message, details: errorDetails(e) }
}

/** `detail` is a line appended after the error's own text; `action` is the button the toast offers. */
export function toastError(
  title: string,
  e: unknown,
  extra: { action?: ToastAction; detail?: string } = {},
): void {
  const details = [errorDetails(e), extra.detail ?? ''].filter((part) => part !== '').join('\n')
  useToasts.getState().push({
    kind: 'error',
    title,
    body: errorMessage(e),
    ...(details === '' ? {} : { detail: details }),
    ...(extra.action === undefined ? {} : { action: extra.action }),
  })
}

export const reportError = (title: string) => (e: unknown) => {
  toastError(title, e)
}

// The sink for a promise nobody awaits. Stores report their own failures, so this only fires for one they did not expect.
export const reportUnexpected = (e: unknown) => {
  toastError(i18n._(msg`Something went wrong`), e)
}
