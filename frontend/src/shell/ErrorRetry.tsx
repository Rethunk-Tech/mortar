import { useLingui } from '@lingui/react/macro'
import { Alert, Button } from '@mui/material'

/** An inline read failure with a Retry button, announced as an alert. */
export function ErrorRetry({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useLingui()
  return (
    <Alert
      severity="error"
      action={
        <Button color="inherit" size="small" onClick={onRetry}>
          {t`Retry`}
        </Button>
      }
    >
      {message}
    </Alert>
  )
}
