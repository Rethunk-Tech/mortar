import { plural } from '@lingui/core/macro'
import { i18n } from './index.ts'

const NAMES_SHOWN = 3

/** Joins names for a sentence in the active locale; past `shown` names the rest collapse into "and N more". total
 * counts names the caller already left out of `names`. */
export function listNames(
  names: readonly string[],
  shown = NAMES_SHOWN,
  total = names.length,
): string {
  const format = new Intl.ListFormat(i18n.locale, { type: 'conjunction' })
  if (total <= shown) {
    return format.format(names)
  }
  const rest = total - shown
  return format.format([...names.slice(0, shown), plural(rest, { one: '# more', other: '# more' })])
}
