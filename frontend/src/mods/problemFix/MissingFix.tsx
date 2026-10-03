import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { useMods } from '../store.ts'
import { ListedFix } from './ListedFix.tsx'
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
  const restoreDismissed = useMods((s) => s.restoreDismissed)
  const dismissListed = useMods((s) => s.dismissListed)
  const { missing } = problem
  if (dismissedToken !== undefined) {
    return (
      <>
        {missing.where ? <WhereButtons where={missing.where} addLabel={t`Add`} /> : null}
        <Button
          size="small"
          color="inherit"
          variant="text"
          onClick={() => restoreDismissed(dismissedToken).catch(reportUnexpected)}
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Restore`}
        </Button>
      </>
    )
  }
  if (missing.listed) {
    return <ListedFix problem={problem} dismiss={dismissListed} />
  }
  if (missing.reason === 'disabled') {
    const off = mods.find((m) => !m.enabled && sameId(m.uniqueId, missing.uniqueId))
    return off ? button(t`Switch on`, () => setEnabled(off, true).catch(reportUnexpected)) : null
  }
  const { where } = missing
  if (!where) {
    return null
  }
  return <WhereButtons where={where} addLabel={t`Add`} />
}
