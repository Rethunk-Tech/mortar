import { useLingui } from '@lingui/react/macro'
import { SetModEnabled } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { openTarget } from '../storeView.ts'
import type { WarningButton } from './warningButton.tsx'

// Keeps the copy with the newest plugin version and switches the other packages off.
export function PluginClashFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'pluginClash' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const { copies, keep } = problem.pluginClash
  return button(t`Keep newer`, () => {
    const open = openTarget()
    if (!open) {
      return
    }
    Promise.all(
      (copies ?? [])
        .filter((copy) => copy.key !== keep)
        .map((copy) => SetModEnabled(open.game, open.id, copy.key, copy.id, false)),
    )
      .then(() => useMods.getState().load())
      .catch(reportUnexpected)
  })
}
