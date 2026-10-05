import { msg, plural } from '@lingui/core/macro'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/models.ts'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'

interface ImportBatch {
  name: string
  watch: BatchWatch
  pendingSettings: number
  showFailed: (batchId: string) => void
}

const batches = new Map<string, ImportBatch>()
const settled = new Set(['done', 'failed', 'skipped', 'cancelled'])

function importCountsLine(counts: SettledImportCounts) {
  const count = counts.installed
  return [
    counts.installed > 0 &&
      i18n._(
        msg`${plural(count, { one: '# mod installed', other: '# mods installed' })}`,
      ),
    counts.failed > 0 && i18n._(msg`${counts.failed} failed`),
    counts.skipped > 0 && i18n._(msg`${counts.skipped} skipped`),
  ]
    .filter(Boolean)
    .join(', ')
}

interface SettledImportCounts {
  installed: number
  failed: number
  skipped: number
}

type Row = Pick<Item, 'id' | 'state'>

/** Reads a batch's queue snapshots and returns its counts once every item has settled. */
type BatchWatch = (items: readonly Row[]) => SettledImportCounts | undefined

/**
 * Follows a batch by the ids of the queue items Add returned. The queue drops its oldest finished items, so an
 * item that was listed and then is gone counts as the last state seen, or as skipped when it went unfinished.
 */
export function watchBatch(ids: readonly string[]): BatchWatch {
  const want = new Set(ids)
  const last = new Map<string, string>()
  return (items) => {
    const listed = new Set<string>()
    for (const item of items) {
      if (want.has(item.id)) {
        last.set(item.id, item.state)
        listed.add(item.id)
      }
    }
    const states: string[] = []
    for (const id of want) {
      const state = last.get(id)
      if (state === undefined) {
        return
      }
      states.push(listed.has(id) || settled.has(state) ? state : 'skipped')
    }
    if (states.some((state) => !settled.has(state))) {
      return
    }
    const installed = states.filter((state) => state === 'done').length
    const failed = states.filter((state) => state === 'failed').length
    return { installed, failed, skipped: states.length - installed - failed }
  }
}

export function trackImport(
  result: Result,
  showFailed: (batchId: string) => void,
  pendingSettings: number,
  current?: readonly Row[],
) {
  if (!(result.batchId && result.items?.length)) {
    return
  }
  batches.set(result.batchId, {
    name: result.profile.name,
    watch: watchBatch(result.items),
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

export function observeImportState(items: readonly Row[]) {
  for (const [batchId, batch] of batches) {
    const counts = batch.watch(items)
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
