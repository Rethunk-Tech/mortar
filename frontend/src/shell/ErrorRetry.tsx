import { useLingui } from '@lingui/react/macro'
import { Alert, Button } from '@mui/material'
import type { InlineError } from '../toasts/report.ts'
import { SignInPrompt } from './SignInPrompt.tsx'

/** An inline read failure with a Retry button, announced as an alert. A missing sign-in shows the sign-in prompt for
 * `source` instead, saying `signedOut` when given. */
export function ErrorRetry({
  error,
  onRetry,
  source,
  signedOut,
}: {
  error: InlineError
  onRetry: () => void
  source?: string | undefined
  signedOut?: string | undefined
}) {
  const { t } = useLingui()
  if (error.signedOut) {
    return <SignInPrompt source={source}>{signedOut}</SignInPrompt>
  }
  return (
    <Alert
      variant="compact"
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
