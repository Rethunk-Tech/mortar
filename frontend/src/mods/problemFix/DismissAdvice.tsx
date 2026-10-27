import { useLingui } from '@lingui/react/macro'
import { EyeOff } from 'lucide-react'
import { reportUnexpected } from '../../toasts/report.ts'
import { useMods } from '../store.ts'
import { FixAction } from './fixActions.tsx'
import { RestoreButton } from './RestoreButton.tsx'

// Dismisses, or once dismissed restores, the advice about a mod's standing: one dismissal covers its broken,
// deprecated and compatibility-list rows.
export function DismissAdvice({
  id,
  dismissedToken,
}: {
  id: string
  dismissedToken?: string | undefined
}) {
  const { t } = useLingui()
  const dismissAbandoned = useMods((s) => s.dismissAbandoned)
  if (dismissedToken !== undefined) {
    return <RestoreButton token={dismissedToken} />
  }
  return (
    <FixAction
      icon={<EyeOff size={16} aria-hidden={true} />}
      label={t`Dismiss`}
      weight="quiet"
      onClick={() => dismissAbandoned(id).catch(reportUnexpected)}
    />
  )
}
