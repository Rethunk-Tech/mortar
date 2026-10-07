import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { profileCardLastPlayedIso } from '../games/profileCardLastPlayed.ts'
import { useSettings } from '../settings/store.ts'
import { useLoaded } from '../shell/useLoaded.ts'

// When the profile's latest run started ('' when it never ran); re-read after each run, which moves `updated`.
export function useLastRun(game: string, profileId: string, updated: string): string {
  const played = useSettings((s) => s.lastPlayed?.[game])
  const { data: started } = useLoaded(
    () => Runs(game, profileId).then((runs) => runs?.[0]?.started ?? ''),
    [game, profileId, updated],
    '',
  )
  return profileCardLastPlayedIso(profileId, played, { [profileId]: started })
}
