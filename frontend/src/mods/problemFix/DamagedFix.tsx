import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Repair } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/storecheck/service.ts'
import { i18n } from '../../i18n/index.ts'
import { reportError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { openTarget } from '../storeView.ts'
import type { WarningButton } from './warningButton.tsx'

// Repair fetches the stored copy again from where the mod came from; a download goes through the queue.
export function DamagedFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'damaged' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const { name, key } = problem.damaged
  const repair = () => {
    const at = openTarget()
    if (!at) {
      return
    }
    Repair(at.game, at.id, key)
      .then((result) => {
        useToasts.getState().push({
          kind: 'info',
          title:
            result.status === 'queued'
              ? i18n._(msg`Downloading ${name} again`)
              : i18n._(msg`Repaired ${name}`),
        })
        return useMods.getState().loadProblems()
      })
      .catch(reportError(t`Could not repair ${name}`))
  }
  return button(t`Repair`, repair)
}
