import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { ReactNode } from 'react'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { openPage } from '../menu.ts'
import { useMods } from '../store.ts'
import { DismissAdvice } from './DismissAdvice.tsx'
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
        <WhereButtons where={where} addLabel={t`Replace with ${{ name: replaceName }}`} />
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
  const advice =
    dismissedToken !== undefined ||
    broken.status === 'abandoned' ||
    broken.status === 'obsolete' ||
    broken.status === 'deprecated' ||
    (broken.source ?? '') !== ''
  const dismiss = advice ? <DismissAdvice id={broken.id} dismissedToken={dismissedToken} /> : null
  return (
    <>
      {replace}
      {mod
        ? button(t`Disable`, () => setEnabled(mod, false).catch(reportUnexpected), Boolean(replace))
        : null}
      {dismiss}
    </>
  )
}
