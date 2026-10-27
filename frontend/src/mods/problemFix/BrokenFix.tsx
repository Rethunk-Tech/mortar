import { useLingui } from '@lingui/react/macro'
import { ExternalLink } from 'lucide-react'
import type { ReactNode } from 'react'
import { reportUnexpected } from '../../toasts/report.ts'
import { type Problem, sameId } from '../lookup.ts'
import { openPage } from '../menu.ts'
import { useMods } from '../store.ts'
import { DismissAdvice } from './DismissAdvice.tsx'
import { FixAction } from './fixActions.tsx'
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
        <FixAction
          icon={<ExternalLink size={16} aria-hidden={true} />}
          label={t`Open replacement`}
          weight="main"
          onClick={() => openPage(where.url)}
        />
      ) : (
        <WhereButtons where={where} addLabel={t`Replace with ${{ name: replaceName }}`} />
      )
  } else if (where?.url) {
    replace = (
      <FixAction
        icon={<ExternalLink size={16} aria-hidden={true} />}
        label={t`Open page`}
        weight="main"
        onClick={() => openPage(where.url)}
      />
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
