import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'

/** shareInFlight runs read once per key while a read for that key is under way, and keeps nothing once it settles. */
export function shareInFlight<T>(): (key: string, read: () => Promise<T>) => Promise<T> {
  const inFlight = new Map<string, Promise<T>>()
  return (key, read) => {
    const pending = inFlight.get(key)
    if (pending !== undefined) {
      return pending
    }
    const started = read().finally(() => inFlight.delete(key))
    inFlight.set(key, started)
    return started
  }
}

const shared = shareInFlight<Profile[] | null>()

/** A game's profiles. One Home load asks from many places at once and each answer carries every profile's entries, so
 * callers asking while a read is under way share it. */
export function listProfiles(game: string): Promise<Profile[] | null> {
  return shared(game, () => Promise.resolve(List(game)))
}
