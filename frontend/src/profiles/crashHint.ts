import type { CrashHint as Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { CrashHint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'

// The profile's updated time moves after each run, when a new hint may exist. A hint that cannot be read shows none.
export function useCrashHint(game: string, profileId: string, updated: string): Hint | null {
  return useLoaded(() => CrashHint(game, profileId), [game, profileId, updated], null).data
}
