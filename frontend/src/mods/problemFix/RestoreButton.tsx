import { useLingui } from '@lingui/react/macro'
import { reportUnexpected } from '../../toasts/report.ts'
import { useMods } from '../store.ts'
import { FixAction } from './fixActions.tsx'

export function RestoreButton({ token }: { token: string }) {
  const { t } = useLingui()
  const restoreDismissed = useMods((s) => s.restoreDismissed)
  return (
    <FixAction
      label={t`Restore`}
      weight="quiet"
      onClick={() => restoreDismissed(token).catch(reportUnexpected)}
    />
  )
}
