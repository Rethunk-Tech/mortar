import { Refresh } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { createSignal } from '../shell/signal.ts'

const refreshed = createSignal()

/** Runs the open Browse query again. */
export const onBrowseRefresh = refreshed.subscribe

/** Drops the game's cached results and rebuilt listings, then has the open Browse search again. */
export async function refreshBrowse(game: string): Promise<void> {
  await Refresh(game)
  refreshed.emit()
}
