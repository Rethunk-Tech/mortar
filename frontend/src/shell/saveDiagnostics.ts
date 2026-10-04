import { msg } from '@lingui/core/macro'
import { SaveDiagnostics } from '../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// Asks where to write a redacted diagnostics zip; game and profile may be ''. Cancel returns no path.
export function saveDiagnostics(game: string, profile: string): void {
  SaveDiagnostics(game, profile)
    .then((path) => {
      if (typeof path === 'string' && path !== '') {
        useToasts.getState().push({
          kind: 'success',
          title: i18n._(msg`Diagnostics saved`),
          body: path,
        })
      }
    })
    .catch(reportUnexpected)
}
