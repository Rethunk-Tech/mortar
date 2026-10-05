import { useLingui } from '@lingui/react/macro'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { useMods } from '../store.ts'
import { ListedFix } from './ListedFix.tsx'
import { RestoreButton } from './RestoreButton.tsx'
import { WhereButtons } from './WhereButtons.tsx'
import type { WarningButton } from './warningButton.tsx'

export function MissingFix({
  problem,
  dismissedToken,
  button,
}: {
  problem: Extract<Problem, { kind: 'missing' }>
  dismissedToken?: string | undefined
  button: WarningButton
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const dismissListed = useMods((s) => s.dismissListed)
  const { missing } = problem
  if (dismissedToken !== undefined) {
    return (
      <>
        {missing.where ? <WhereButtons where={missing.where} addLabel={t`Add`} /> : null}
        <RestoreButton token={dismissedToken} />
      </>
    )
  }
  if (missing.listed) {
    return <ListedFix problem={problem} dismiss={dismissListed} />
  }
  if (missing.reason === 'disabled') {
    const off = mods.find((m) => !m.enabled && sameId(m.id, missing.id))
    return off ? button(t`Enable`, () => setEnabled(off, true).catch(reportUnexpected)) : null
  }
  const { where } = missing
  if (!where) {
    return null
  }
  return <WhereButtons where={where} addLabel={t`Add`} />
}
