import { msg } from '@lingui/core/macro'
import type { EverywhereFailed } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { useToasts } from '../toasts/store.ts'

const reasonText = (reason: string) => {
  if (reason === 'locked') {
    return i18n._(msg`game running`)
  }
  if (reason === 'needs install choices') {
    return i18n._(msg`needs install choices`)
  }
  return reason
}

// An update that downloaded first could not reach some other profiles: name them and why.
export function reportEverywhereFailed({ mod, failed }: EverywhereFailed) {
  const rows = failed ?? []
  const named = rows.filter((f) => f.profile !== '')
  useToasts.getState().push({
    kind: 'warning',
    title:
      named.length > 0
        ? i18n._(
            msg`Could not update ${{ mod }} in ${{ profiles: listNames(named.map((f) => f.profile)) }}`,
          )
        : i18n._(msg`Could not update ${{ mod }} in the other profiles`),
    body: rows
      .map((f) => (f.profile ? `${f.profile}: ${reasonText(f.reason)}` : reasonText(f.reason)))
      .join('\n'),
  })
}
