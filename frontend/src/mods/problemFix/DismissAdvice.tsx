import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { reportUnexpected } from '../../toasts/report.ts'
import { useMods } from '../store.ts'
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
    <Button
      size="small"
      color="inherit"
      variant="text"
      onClick={() => dismissAbandoned(id).catch(reportUnexpected)}
      sx={{ flexShrink: 0 }}
    >
      {t`Dismiss`}
    </Button>
  )
}
