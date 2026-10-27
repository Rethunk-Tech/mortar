import { useLingui } from '@lingui/react/macro'
import { EyeOff } from 'lucide-react'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { FixAction } from './fixActions.tsx'
import { WhereButtons } from './WhereButtons.tsx'

export function ListedFix({
  problem,
  dismiss,
}: {
  problem: Extract<Problem, { kind: 'missing' }>
  dismiss: (id: string) => Promise<void>
}) {
  const { t } = useLingui()
  const { missing } = problem
  return (
    <>
      {missing.where ? <WhereButtons where={missing.where} addLabel={t`Add`} /> : null}
      <FixAction
        icon={<EyeOff size={16} aria-hidden={true} />}
        label={t`Dismiss`}
        weight="quiet"
        onClick={() => dismiss(missing.id).catch(reportUnexpected)}
      />
    </>
  )
}
