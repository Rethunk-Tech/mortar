import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { ReactNode } from 'react'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { openPage } from '../menu.ts'
import { useMods } from '../store.ts'
import { RestoreButton } from './RestoreButton.tsx'
import { WhereButtons } from './WhereButtons.tsx'
import type { WarningButton } from './warningButton.tsx'

export function BrokenFix({
  problem,
  dismissedToken,
  button,
}: {
  problem: Extract<Problem, { kind: 'broken' }>
  dismissedToken?: string | undefined
  button: WarningButton
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const dismissAbandoned = useMods((s) => s.dismissAbandoned)
  const { broken } = problem
  const mod = mods.find((m) => m.key === broken.key && sameId(m.id, broken.id))
  const where = broken.replacement
  const replaceName =
    where?.pageName?.trim() ||
    where?.github?.trim() ||
    where?.fileName?.trim() ||
    (where && where.pageId > 0 ? String(where.pageId) : '')
  let replace: ReactNode = null
  if (where && replaceName !== '') {
    replace =
      (broken.status === 'obsolete' || broken.status === 'deprecated') && where.url ? (
        <Button
          size="small"
          color="warning"
          variant="outlined"
          onClick={() => openPage(where.url)}
          sx={{ flexShrink: 0 }}
        >
          {t`Open replacement`}
        </Button>
      ) : (
        <WhereButtons where={where} addLabel={t`Replace with ${replaceName}`} />
      )
  } else if (where?.url) {
    replace = (
      <Button
        size="small"
        color="warning"
        variant="contained"
        onClick={() => openPage(where.url)}
        sx={{ flexShrink: 0 }}
      >
        {t`Open page`}
      </Button>
    )
  }
  let dismiss: ReactNode = null
  if (dismissedToken !== undefined) {
    dismiss = <RestoreButton token={dismissedToken} />
  } else if (
    broken.status === 'abandoned' ||
    broken.status === 'obsolete' ||
    broken.status === 'deprecated'
  ) {
    dismiss = (
      <Button
        size="small"
        color="inherit"
        variant="text"
        onClick={() => dismissAbandoned(broken.id).catch(reportUnexpected)}
        sx={{ flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    )
  }
  return (
    <>
      {replace}
      {mod
        ? button(
            t`Switch off`,
            () => setEnabled(mod, false).catch(reportUnexpected),
            Boolean(replace),
          )
        : null}
      {dismiss}
    </>
  )
}
