import { TestMods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { queueWants } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'

const FINISHED = new Set(['done', 'failed', 'skipped', 'cancelled'])

/** Installs the mods the profile's projects test with that it lacks, and returns once they have finished. */
export async function installTestMods(game: string, profile: string) {
  const missing = (await TestMods(game, profile)) ?? []
  const items =
    missing.length > 0
      ? await queueWants(missing.map((id) => ({ kind: 'install', package: id })))
      : undefined
  if (!items) {
    return
  }
  const ids = new Set(items.map((i) => i.id))
  const finished = () =>
    useQueue
      .getState()
      .state.items.filter((i) => ids.has(i.id))
      .every((i) => FINISHED.has(i.state))
  if (finished()) {
    return
  }
  await new Promise<void>((resolve) => {
    const stop = useQueue.subscribe(() => {
      if (finished()) {
        stop()
        resolve()
      }
    })
  })
}
