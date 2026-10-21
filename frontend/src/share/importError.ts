import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'
import { errorKind } from '../toasts/errorKind.ts'
import { errorMessage } from '../toasts/report.ts'

// What went wrong with the link, code or file the import dialog was given, said about that input.
export function importErrorMessage(e: unknown): string {
  switch (errorKind(e)) {
    case 'invalid':
      return i18n._(
        msg`Mortar doesn't recognise this. Paste a Mortar link, a Nexus collection, or an r2modman or Gale code.`,
      )
    case 'damaged':
      return i18n._(
        msg`Mortar can't read this. Check that the whole link, code or file came through.`,
      )
    case 'not_found':
      return i18n._(
        msg`Nothing was found for that link or code. Codes expire, so ask for a new one.`,
      )
    default:
      return errorMessage(e)
  }
}
