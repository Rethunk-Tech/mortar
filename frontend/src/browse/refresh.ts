import { Refresh } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'

type Listener = () => void

const listeners = new Set<Listener>()

/** Runs the open Browse query again. */
export function onBrowseRefresh(fn: Listener): () => void {
  listeners.add(fn)
  return () => {
    listeners.delete(fn)
  }
}

/** Drops the game's cached results and rebuilt listings, then has the open Browse search again. */
export async function refreshBrowse(game: string): Promise<void> {
  await Refresh(game)
  for (const fn of listeners) {
    fn()
  }
}
