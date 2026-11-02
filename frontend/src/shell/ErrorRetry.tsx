import { useLingui } from '@lingui/react/macro'
import { Alert, Button } from '@mui/material'
import { openSettings } from '../nav/store.ts'
import type { InlineError } from '../toasts/report.ts'

/** An inline read failure with a Retry button, announced as an alert. A missing sign-in offers Settings instead. */
export function ErrorRetry({ error, onRetry }: { error: InlineError; onRetry: () => void }) {
  const { t } = useLingui()
  return (
    <Alert
      variant="compact"
      severity="error"
      title={error.details}
      action={
        error.signedOut ? (
          <Button color="inherit" size="small" onClick={() => openSettings('accounts')}>
            {t`Open Settings`}
          </Button>
        ) : (
          <Button color="inherit" size="small" onClick={onRetry}>
            {t`Retry`}
          </Button>
        )
      }
    >
      {error.message}
    </Alert>
  )
}
