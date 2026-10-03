import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import type { ReactNode } from 'react'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { useMods } from '../store.ts'
import { WhereButtons } from './WhereButtons.tsx'
import type { WarningButton } from './warningButton.tsx'

export function DismissedBrokenFix({
  problem,
  dismissedToken,
  button,
}: {
  problem: Extract<Problem, { kind: 'broken' }>
  dismissedToken: string
  button: WarningButton
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const restoreDismissed = useMods((s) => s.restoreDismissed)
  const mod = mods.find(
    (m) => m.key === problem.broken.key && sameId(m.uniqueId, problem.broken.uniqueId),
  )
  const { replacement } = problem.broken
  let replacementAction: ReactNode = null
  if (
    replacement &&
    problem.broken.status !== 'obsolete' &&
    problem.broken.status !== 'deprecated'
  ) {
    replacementAction = <WhereButtons where={replacement} addLabel={t`Replace`} />
  } else if (replacement?.url) {
    replacementAction = (
      <Button
        size="small"
        color="warning"
        variant="outlined"
        onClick={() => Browser.OpenURL(replacement.url ?? '').catch(reportUnexpected)}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Open replacement`}
      </Button>
    )
  }
  return (
    <>
      {mod ? button(t`Switch off`, () => setEnabled(mod, false).catch(reportUnexpected)) : null}
      {replacementAction}
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
