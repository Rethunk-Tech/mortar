import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { RotateCcw } from 'lucide-react'
import type { RetryAllResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import { RetryAllFailed } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

// The skipped reasons of a retry-all as one line, "2 already queued, 1 superseded"; '' when nothing was skipped.
function skippedSummary(skipped: RetryAllResult['skipped']): string {
  const queued = skipped?.queued ?? 0
  const superseded = skipped?.superseded ?? 0
  const incomplete = skipped?.incomplete ?? 0
  return [
    queued > 0 ? plural(queued, { one: '# already queued', other: '# already queued' }) : '',
    superseded > 0
      ? plural(superseded, {
          one: '# superseded by a newer download',
          other: '# superseded by a newer download',
        })
      : '',
    incomplete > 0
      ? plural(incomplete, { one: '# incomplete entry', other: '# incomplete entries' })
      : '',
  ]
    .filter((s) => s !== '')
    .join(', ')
}

function retriedTitle(requeued: number): string {
  return plural(requeued, { one: 'Retried # download', other: 'Retried # downloads' })
}

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
      >
        {t`Retry all failed`}
      </Button>
    </DisabledReason>
  )
}
