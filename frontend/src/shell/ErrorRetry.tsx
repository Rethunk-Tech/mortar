import { useLingui } from '@lingui/react/macro'
import { Alert, Button } from '@mui/material'
import type { InlineError } from '../toasts/report.ts'

/** An inline read failure with a Retry button, announced as an alert. */
export function ErrorRetry({ error, onRetry }: { error: InlineError; onRetry: () => void }) {
  const { t } = useLingui()
  return (
    <Alert
      severity="error"
      title={error.details}
      action={
        <Button color="inherit" size="small" onClick={onRetry}>
          {t`Retry`}
        </Button>
      }
    >
      {error.message}
    </Alert>
  )
}
