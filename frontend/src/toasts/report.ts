import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { useToasts } from './store.ts'

// The sink for a promise nobody awaits. Stores report their own failures, so this only fires for one they did not expect.
export const reportUnexpected = (e: unknown) => {
  useToasts
    .getState()
    .push({ kind: 'error', title: i18n._(msg`Something went wrong`), body: String(e) })
}

export function errorText(e: unknown): string | undefined {
  if (e instanceof Error) {
    return e.message
  }
  return typeof e === 'string' ? e : undefined
}
