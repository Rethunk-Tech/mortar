import { msg } from '@lingui/core/macro'
import { i18n } from '../../i18n/index.ts'
import { errorDetails } from '../../toasts/errorKind.ts'
import { errorMessage } from '../../toasts/report.ts'

// The backend refuses a value with `config_value:<problem>[:<min>:<max>] <english>`; "any" stands for a missing bound.
const PREFIX = 'config_value:'

/** The sentence for a refused config value, or the error's generic sentence when it is not one. */
export function configValueMessage(e: unknown): string {
  const details = errorDetails(e)
  if (!details.startsWith(PREFIX)) {
    return errorMessage(e)
  }
  const [problem, min, max] = (details.slice(PREFIX.length).split(' ', 1)[0] ?? '').split(':')
  switch (problem) {
    case 'whole_number':
      return i18n._(msg`This needs to be a whole number.`)
    case 'number':
      return i18n._(msg`This needs to be a number.`)
    case 'bool':
      return i18n._(msg`This needs to be true or false.`)
    case 'enum':
      return i18n._(msg`This is not one of the allowed values.`)
    case 'range':
      if (min !== undefined && max !== undefined) {
        if (min === 'any') {
          return i18n._(msg`This can be at most ${max}.`)
        }
        if (max === 'any') {
          return i18n._(msg`This can be at least ${min}.`)
        }
        return i18n._(msg`This needs to be between ${min} and ${max}.`)
      }
      return errorMessage(e)
    default:
      return errorMessage(e)
  }
}
