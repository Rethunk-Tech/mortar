import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
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
      <Button
        size="small"
        color="inherit"
        variant="text"
        onClick={() => dismiss(missing.id).catch(reportUnexpected)}
        sx={{ flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    </>
  )
}
