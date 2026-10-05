import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import type { HistoryEvent } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  History,
  Revert,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useTab } from '../../game/tab.ts'
import { i18n } from '../../i18n/index.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useToasts } from '../../toasts/store.ts'

export interface UndoAllTarget {
  game: string
  profileId: string
  /** The history event holding the profile as it stood before the batch. */
  beforeId: string
  /** Changes made after the batch, which restoring would revert too. */
  later: HistoryEvent[]
}

/** Events after the restore point that the batch itself did not make: the batch records one bulk event (or one
 * update event for a single mod) right after the restore point. History lists newest first. */
export function changesAfterBatch(
  events: readonly HistoryEvent[],
  beforeId: string,
): HistoryEvent[] {
  const at = events.findIndex((e) => e.id === beforeId)
  const after = (at < 0 ? [] : events.slice(0, at)).toReversed()
  const [first] = after
  return first && (first.kind === 'bulk' || first.kind === 'updated') ? after.slice(1) : after
}

export const useUndoAll = create<{ target: UndoAllTarget | null }>(() => ({ target: null }))

export async function restoreBatch(game: string, profileId: string, beforeId: string) {
  useProfiles.getState().replace(await Revert(game, profileId, beforeId))
  useToasts.getState().push({
    kind: 'success',
    title: i18n._(msg`Restored the profile from before Update all`),
    body: i18n._(msg`Downloaded versions stay in the store for re-use.`),
  })
}

/** Restores the snapshot taken before Update all; asks first when the profile changed after the batch. */
export async function undoAll(game: string, profileId: string, beforeId: string) {
  const later = changesAfterBatch((await History(game, profileId)) ?? [], beforeId)
  if (later.length === 0) {
    await restoreBatch(game, profileId, beforeId)
    return
  }
  useTab.getState().setTab('mods')
  useUndoAll.setState({ target: { game, profileId, beforeId, later } })
}
