import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'

/** The label of a known-broken list's kind: Broken, Superseded or Not recommended. */
export function brokenKindLabel(kind: string | undefined): string {
  if (kind === 'superseded') {
    return i18n._(msg`Superseded`)
  }
  if (kind === 'notRecommended') {
    return i18n._(msg`Not recommended`)
  }
  return i18n._(msg`Broken`)
}
