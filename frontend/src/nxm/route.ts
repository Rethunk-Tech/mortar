import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import type { Route } from '../nav/store.ts'

// The game an nxm link for stardewvalley belongs to.
export const NXM_GAME = 'stardew'

// The profile a link installs into without asking: the open profile while its game's screen is showing. Any other
// screen (game select, settings, first run, profiles page) has no such profile and the user is asked.
export function directProfile(
  route: Route,
  gameId: string | undefined,
  openId: string,
  profiles: Profile[],
): Profile | null {
  if (route.name !== 'game' || route.game !== NXM_GAME || gameId !== NXM_GAME) {
    return null
  }
  return profiles.find((p) => p.id === openId) ?? null
}
