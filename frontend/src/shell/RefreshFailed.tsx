import { useLingui } from '@lingui/react/macro'
import { Alert, Button } from '@mui/material'
import { formatWhen } from '../i18n/formatWhen.ts'
import { openSettings } from '../nav/store.ts'
import type { InlineError } from '../toasts/report.ts'

/** One line above data a view keeps showing from its last good copy after a refresh failed. Nothing when `error` is
 * null. A missing sign-in offers signing in instead of retrying. */
export function RefreshFailed({
  error,
  at,
  onRetry,
}: {
  error: InlineError | null
  at?: number | null | undefined
  onRetry: () => unknown
}) {
  const { t } = useLingui()
  if (error === null) {
    return null
  }
  if (error.signedOut) {
    return (
      <Alert
        variant="compact"
        severity="info"
        action={
          <Button color="inherit" size="small" onClick={() => openSettings('accounts')}>
            {t`Sign in…`}
          </Button>
        }
      >
        {at
          ? t`Saved ${{ when: formatWhen(at) }}. Sign in to refresh it.`
          : t`Saved from an earlier visit. Sign in to refresh it.`}
      </Alert>
    )
  }
  return (
    <Alert
      variant="compact"
      severity="warning"
      title={error.details}
      action={
        <Button color="inherit" size="small" onClick={() => onRetry()}>
          {t`Retry`}
        </Button>
      }
    >
      {at
        ? t`Could not refresh. Showing what was read ${{ when: formatWhen(at) }}.`
        : t`Could not refresh, showing the last copy`}
    </Alert>
  )
}
