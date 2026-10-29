import { msg } from '@lingui/core/macro'
import { i18n } from '../../i18n/index.ts'
import { errorMessage } from '../../toasts/report.ts'

/** The refusal's `detail` (configsvc's valueError); min and max are absent for a bound the setting does not have. */
interface ValueDetail {
  problem: string
  min?: number
  max?: number
}

function detailOf(e: unknown): ValueDetail | null {
  const cause: unknown = e instanceof Error ? e.cause : undefined
  if (typeof cause !== 'object' || cause === null || !('detail' in cause)) {
    return null
  }
  const d = cause.detail
  if (typeof d !== 'object' || d === null || !('problem' in d) || typeof d.problem !== 'string') {
    return null
  }
  const min = 'min' in d && typeof d.min === 'number' ? d.min : undefined
  const max = 'max' in d && typeof d.max === 'number' ? d.max : undefined
  return { problem: d.problem, min, max }
}

/** The sentence for a refused config value, or the error's generic sentence when it is not one. */
export function configValueMessage(e: unknown): string {
  const d = detailOf(e)
  if (d === null) {
    return errorMessage(e)
  }
  switch (d.problem) {
    case 'line':
      return i18n._(msg`This needs to be on one line.`)
    case 'whole_number':
      return i18n._(msg`This needs to be a whole number.`)
    case 'number':
      return i18n._(msg`This needs to be a number.`)
    case 'bool':
      return i18n._(msg`This needs to be true or false.`)
    case 'enum':
      return i18n._(msg`This is not one of the allowed values.`)
    case 'color':
      return i18n._(msg`This needs to be a hex color like FF8800 or FF8800CC.`)
    case 'list':
      return i18n._(msg`This needs to be a list in [brackets].`)
    case 'empty':
      return i18n._(msg`This needs a value.`)
    case 'raw':
      return i18n._(msg`This is not a value the file can hold.`)
    case 'toml':
      return i18n._(msg`This would leave the file unreadable to the game.`)
    case 'range': {
      const n = new Intl.NumberFormat(i18n.locale)
      const min = d.min === undefined ? undefined : n.format(d.min)
      const max = d.max === undefined ? undefined : n.format(d.max)
      if (min !== undefined && max !== undefined) {
        return i18n._(msg`This needs to be between ${min} and ${max}.`)
      }
      if (max !== undefined) {
        return i18n._(msg`This can be at most ${max}.`)
      }
      if (min !== undefined) {
        return i18n._(msg`This can be at least ${min}.`)
      }
      return errorMessage(e)
    }
    default:
      return errorMessage(e)
  }
}
