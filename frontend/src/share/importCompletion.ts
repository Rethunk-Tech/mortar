import { msg, plural } from '@lingui/core/macro'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'

interface ImportBatch {
  name: string
  total: number
  pendingSettings: number
  showFailed: (batchId: string) => void
}

const batches = new Map<string, ImportBatch>()
const settled = new Set(['done', 'failed', 'skipped', 'cancelled'])

function importCountsLine(counts: SettledImportCounts) {
  return [
    counts.installed > 0 && i18n._(msg`${{ what: counts.installed }} installed`),
    counts.failed > 0 && i18n._(msg`${counts.failed} failed`),
    counts.skipped > 0 && i18n._(msg`${counts.skipped} skipped`),
  ]
    .filter(Boolean)
    .join(', ')
}

export interface SettledImportCounts {
  installed: number
  failed: number
  skipped: number
}

export function settledImportCounts(
  items: readonly Pick<Item, 'batchId' | 'state'>[],
  batchId: string,
  total: number,
): SettledImportCounts | undefined {
  const rows = items.filter((item) => item.batchId === batchId)
  if (rows.length < total || rows.some((item) => !settled.has(item.state))) {
    return
  }
  const installed = rows.filter((item) => item.state === 'done').length
  const failed = rows.filter((item) => item.state === 'failed').length
  return { installed, failed, skipped: rows.length - installed - failed }
}

export function trackImport(
  result: Result,
  showFailed: (batchId: string) => void,
  pendingSettings: number,
  current?: readonly Pick<Item, 'batchId' | 'state'>[],
) {
  if (!result.batchId || result.queued <= 0) {
    return
  }
  batches.set(result.batchId, {
    name: result.profile.name,
    total: result.queued,
    pendingSettings,
    showFailed,
  })
  if (current) {
    observeImportState(current)
  }
}

export function isTrackedImportBatch(batchId: string): boolean {
  return batchId !== '' && batches.has(batchId)
}

export function observeImportState(items: readonly Pick<Item, 'batchId' | 'state'>[]) {
  for (const [batchId, batch] of batches) {
    const counts = settledImportCounts(items, batchId, batch.total)
    if (counts) {
      useToasts.getState().push({
        kind: counts.failed > 0 ? 'warning' : 'success',
        title: i18n._(msg`Imported ${batch.name}: ${importCountsLine(counts)}`),
        ...(batch.pendingSettings > 0
          ? {
              body: i18n._(
                msg`${plural(batch.pendingSettings, {
                  one: '# setting is still waiting for its mod to install.',
                  other: '# settings are still waiting for their mods to install.',
                })}`,
              ),
            }
          : {}),
        ...(counts.failed > 0
          ? {
              action: {
                label: i18n._(msg`Show failed`),
                run: () => batch.showFailed(batchId),
              },
            }
          : {}),
      })
      batches.delete(batchId)
    }
  }
}
