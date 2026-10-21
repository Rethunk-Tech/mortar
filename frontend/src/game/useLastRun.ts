import { LastRunStarted } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { profileCardLastPlayedIso } from '../games/profileCardLastPlayed.ts'
import { useSettings } from '../settings/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'

const inFlight = new Map<string, { updated: string; started: Promise<string> }>()

// One read per profile and `updated` stamp, shared by every caller on Home; a failed read is not kept.
export function lastRunStarted(game: string, profileId: string, updated: string): Promise<string> {
  const key = `${game}\0${profileId}`
  const hit = inFlight.get(key)
  if (hit?.updated === updated) {
    return hit.started
  }
  const started = LastRunStarted(game, profileId)
  inFlight.set(key, { updated, started })
  started.catch(() => inFlight.delete(key))
  return started
}

// When the profile's latest run started ('' when it never ran); re-read after each run, which moves `updated`.
export function useLastRun(game: string, profileId: string, updated: string): string {
  const played = useSettings((s) => s.lastPlayed?.[game])
  const { data: started } = useLoaded(
    () => lastRunStarted(game, profileId, updated),
    [game, profileId, updated],
    '',
  )
  return profileCardLastPlayedIso(profileId, played, { [profileId]: started })
}
