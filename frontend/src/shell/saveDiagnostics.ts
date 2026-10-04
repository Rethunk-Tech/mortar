import { msg } from '@lingui/core/macro'
import {
  SaveDiagnostics,
  ShowDiagnostics,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

function showDiagnostics(path: string) {
  return ShowDiagnostics(path).catch(reportUnexpected)
}

// Asks where to write a redacted diagnostics zip; game and profile may be ''. Resolves true once saved, false on cancel
// or failure. forIssue words the toast for a bug report, where the zip is dragged into the GitHub issue.
export function saveDiagnostics(game: string, profile: string, forIssue = false): Promise<boolean> {
  return SaveDiagnostics(game, profile)
    .then((path) => {
      if (typeof path !== 'string' || path === '') {
        return false
      }
      useToasts.getState().push({
        kind: 'success',
        title: i18n._(msg`Diagnostics saved`),
        body: forIssue ? i18n._(msg`Drag ${path} into the GitHub issue to attach it.`) : path,
        action: {
          label: i18n._(msg`Show file`),
          run: () => showDiagnostics(path),
        },
      })
      return true
    })
    .catch((e: unknown) => {
      reportUnexpected(e)
      return false
    })
}
