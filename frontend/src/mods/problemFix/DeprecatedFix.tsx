import { useLingui } from '@lingui/react/macro'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { download } from '../../queue/actions.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { entryOf, type Problem } from '../lookup.ts'
import { DismissAdvice } from './DismissAdvice.tsx'
import type { WarningButton } from './warningButton.tsx'

// Queues the replacement package the deprecated one points to, with its install reason, when it names one.
export function DeprecatedFix({
  problem,
  dismissedToken,
  button,
}: {
  problem: Extract<Problem, { kind: 'deprecated' }>
  dismissedToken?: string | undefined
  button: WarningButton
}) {
  const { t } = useLingui()
  const implicit = Boolean(entryOf(useProfiles(openProfileOf), problem.deprecated.key)?.implicit)
  const replacement = problem.deprecated.replacement ?? ''
  return (
    <>
      {replacement === '' || dismissedToken !== undefined
        ? null
        : button(t`Replace with ${{ name: replacement }}`, () => {
            download([{ kind: 'install', package: replacement, implicit }]).catch(reportUnexpected)
          })}
      <DismissAdvice id={problem.deprecated.id} dismissedToken={dismissedToken} />
    </>
  )
}
