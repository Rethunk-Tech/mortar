import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { errorDetails as detailsOf, errorKind as kindOf, errorText as textOf } from './errorKind.ts'
import { useToasts } from './store.ts'

const sentences = {
  not_found: msg`That item could not be found.`,
  busy: msg`The game is already running.`,
  network: msg`A network request failed.`,
  permission: msg`Mortar does not have permission to do that.`,
  disk_full: msg`The disk is full.`,
  damaged: msg`That data could not be read.`,
  invalid: msg`That request was not valid.`,
  unknown: msg`Something went wrong.`,
} as const

export function errorText(e: unknown): string | undefined {
  return textOf(e)
}

export function errorKind(e: unknown): ReturnType<typeof kindOf> {
  return kindOf(e)
}

export function errorDetails(e: unknown): string {
  return detailsOf(e)
}

/** Plain sentence for a kind; untagged errors keep their text. */
export function errorMessage(e: unknown): string {
  const kind = kindOf(e)
  if (kind === 'unknown') {
    const raw = detailsOf(e)
    return raw === '' ? i18n._(sentences.unknown) : raw
  }
  return i18n._(sentences[kind])
}

// The sink for a promise nobody awaits. Stores report their own failures, so this only fires for one they did not expect.
export const reportUnexpected = (e: unknown) => {
  useToasts
    .getState()
    .push({ kind: 'error', title: i18n._(msg`Something went wrong`), body: errorMessage(e) })
}
