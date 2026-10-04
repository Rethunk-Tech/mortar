import { plural } from '@lingui/core/macro'
import type { RetryAllResult } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'

// The skipped reasons of a retry-all as one line, "2 already queued, 1 superseded"; '' when nothing was skipped.
export function skippedSummary(skipped: RetryAllResult['skipped']): string {
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

export function retriedTitle(requeued: number): string {
  return plural(requeued, { one: 'Retried # download', other: 'Retried # downloads' })
}
