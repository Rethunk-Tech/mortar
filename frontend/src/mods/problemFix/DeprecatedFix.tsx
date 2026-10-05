import { useLingui } from '@lingui/react/macro'
import { download } from '../../queue/actions.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import type { WarningButton } from './warningButton.tsx'

// Queues the replacement package the deprecated one points to; nothing when it names none.
export function DeprecatedFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'deprecated' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const replacement = problem.deprecated.replacement ?? ''
  if (replacement === '') {
    return null
  }
  return button(t`Replace with ${replacement}`, () => {
    download([{ kind: 'install', package: replacement }]).catch(reportUnexpected)
  })
}
