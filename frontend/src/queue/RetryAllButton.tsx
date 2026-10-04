import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { RotateCcw } from 'lucide-react'
import { RetryAllFailed } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { retriedTitle, skippedSummary } from './retryAll.ts'

export function RetryAllButton({ failed }: { failed: number }) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  if (failed === 0) {
    return null
  }
  const retry = () =>
    run(async () => {
      const result = await RetryAllFailed()
      const body = skippedSummary(result.skipped)
      useToasts.getState().push({
        kind: result.requeued > 0 ? 'success' : 'info',
        title: retriedTitle(result.requeued),
        ...(body === '' ? {} : { body }),
      })
    })
  return (
    <DisabledReason title={t`Already retrying failed downloads`} disabled={pending}>
      <Button
        variant="outlined"
        startIcon={<RotateCcw size={16} />}
        disabled={pending}
        onClick={retry}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {t`Retry all failed`}
      </Button>
    </DisabledReason>
  )
}
