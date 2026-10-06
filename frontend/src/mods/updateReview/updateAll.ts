import { msg, plural } from '@lingui/core/macro'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { BeginUpdateBatch } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { RetryFailed } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/queue/service.ts'
import { i18n } from '../../i18n/index.ts'
import { queueWants, type Want } from '../../queue/actions.ts'
import { useQueue } from '../../queue/store.ts'
import { watchBatch } from '../../share/importCompletion.ts'
import { useToasts } from '../../toasts/store.ts'
import { installableUpdate } from '../lookup.ts'
import { undoAll } from './undoAll.ts'

/** Updates that stay on the source the mod came from; the rest switch its source and wait for a choice each. */
const sameSourceUpdates = (list: readonly Update[]): Update[] =>
  list.filter((u) => !u.switch && installableUpdate(u))

const needChoiceUpdates = (list: readonly Update[]): Update[] =>
  list.filter((u) => u.switch && installableUpdate(u))

function summarize(
  at: { game: string; profileId: string; beforeId: string },
  counts: { installed: number; failed: number; changes: string[] },
  needChoice: number,
) {
  const toasts = useToasts.getState()
  const updated = i18n._(
    msg`${plural(counts.installed, { one: 'Updated # mod', other: 'Updated # mods' })}`,
  )
  const parts = [
    updated,
    ...(counts.failed > 0 ? [i18n._(msg`${counts.failed} failed`)] : []),
    ...(needChoice > 0 ? [i18n._(msg`${needChoice} need your choice`)] : []),
  ]
  toasts.push({
    kind: counts.failed > 0 ? 'warning' : 'success',
    title: parts.join('; '),
    changes: counts.changes,
    ...(counts.installed > 0
      ? {
          action: {
            label: i18n._(msg`Undo all`),
            run: () => undoAll(at.game, at.profileId, at.beforeId),
          },
        }
      : {}),
  })
  if (counts.failed > 0) {
    toasts.push({
      kind: 'warning',
      title: i18n._(
        msg`${plural(counts.failed, { one: '# update failed', other: '# updates failed' })}`,
      ),
      action: { label: i18n._(msg`Retry`), run: () => RetryFailed() },
    })
  }
}

/** Queues every want in one batch after a restore point, and when the batch settles says how it went. */
async function updateAll(
  game: string,
  profileId: string,
  wants: Want[],
  needChoice: number,
): Promise<boolean> {
  const { before: beforeId, batch: batchId } = await BeginUpdateBatch(game, profileId)
  const items = await queueWants(
    wants.map((w) => ({ ...w, batchId })),
    true,
  )
  if (!items) {
    return false
  }
  const watch = watchBatch(items.map((item) => item.id))
  const stop = useQueue.subscribe((s) => {
    const counts = watch(s.state.items)
    if (counts) {
      stop()
      summarize({ game, profileId, beforeId }, counts, needChoice)
    }
  })
  return true
}

export { needChoiceUpdates, sameSourceUpdates, updateAll }
